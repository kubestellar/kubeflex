/*
Copyright 2023 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"context"
	"fmt"
	"log"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/go-logr/logr"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/strvals"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	repoFile = "helm.repo"
)

type HelmHandler struct {
	URL         string
	RepoName    string
	ChartName   string
	ReleaseName string
	Namespace   string
	// version is only used for "classic" helm charts, not OCI
	Version  string
	Args     map[string]string
	log      logr.Logger
	settings *cli.EnvSettings
}

func Init(ctx context.Context, handler *HelmHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	tmpDir := os.TempDir()
	handler.log = clog.FromContext(ctx)
	handler.settings = cli.New()
	handler.settings.SetNamespace(handler.Namespace)
	handler.settings.RepositoryConfig = filepath.Join(tmpDir, repoFile)
	return nil
}

func (h *HelmHandler) Install() error {
	if isOCIURL(h.URL) {
		if err := h.installOCIChart(); err != nil {
			return err
		}
		return nil
	}

	if err := h.repoAdd(); err != nil {
		return err
	}

	if err := h.repoUpdate(); err != nil {
		return err
	}

	if err := h.chartInstall(); err != nil {
		return err
	}

	return nil
}

// RepoAdd adds repo with given name and url
func (h *HelmHandler) repoAdd() error {
	repoFile := h.settings.RepositoryConfig

	//Ensure the file directory exists as it is required for file locking
	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("error initializing dir for helm setup %s", err)
	}

	// Acquire a file lock for process synchronization
	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
	if err == nil && locked {
		defer fileLock.Unlock()
	}
	if err != nil {
		return fmt.Errorf("error acquiring file lock for helm setup %s", err)
	}

	b, err := os.ReadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error opening repo file %s", err)
	}

	var f repo.File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("error unmarshaling %s", err)
	}

	if f.Has(h.RepoName) {
		h.log.V(3).Info("repo already exists", "repoName", h.RepoName)
	}

	c := repo.Entry{
		Name: h.RepoName,
		URL:  h.URL,
	}

	r, err := repo.NewChartRepository(&c, getter.All(h.settings))
	if err != nil {
		log.Fatal(err)
	}

	if _, err := r.DownloadIndexFile(); err != nil {
		err := errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", h.URL)
		return err
	}

	f.Update(&c)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		log.Fatal(err)
	}
	h.log.V(3).Info("repo has been added to your repositories", "name", h.RepoName)
	return nil
}

func (h *HelmHandler) repoUpdate() error {
	repoFile := h.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(errors.Cause(err)) || len(f.Repositories) == 0 {
		return errors.New("no repositories found. You must add one before updating")
	}
	var repos []*repo.ChartRepository
	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(h.settings))
		if err != nil {
			log.Fatal(err)
		}
		repos = append(repos, r)
	}

	h.log.V(3).Info("getting the latest from chart repositories")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				h.log.Error(err, "...Unable to get an update from the chart repository", "name", re.Config.Name, "url", re.Config.URL)
			} else {
				h.log.V(3).Info("successfully got an update from the chart repository", "name", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	h.log.V(3).Info("update complete!")
	return nil
}

func (h *HelmHandler) chartInstall() error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		return err
	}
	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}

	// set the version if specified, otherwise default to latest version
	if h.Version != "" {
		client.ChartPathOptions.Version = h.Version
	}

	client.ReleaseName = h.ReleaseName
	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", h.RepoName, h.ChartName), h.settings)
	if err != nil {
		return err
	}

	h.log.V(3).Info("chart path", "path", cp)

	p := getter.All(h.settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}

	// Add args
	if err := strvals.ParseInto(h.Args["set"], vals); err != nil {
		return errors.Wrap(err, "failed parsing --set data")
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: h.settings.RepositoryConfig,
					RepositoryCache:  h.settings.RepositoryCache,
				}
				if err := man.Update(); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	client.Namespace = h.settings.Namespace()
	release, err := client.Run(chartRequested, vals)
	if err != nil {
		return err
	}
	h.log.V(3).Info(release.Manifest)
	return nil
}

func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
	log.Output(2, fmt.Sprintf(format, v...))
}

func (h *HelmHandler) CheckStatus() (*release.Release, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		log.Fatal(err)
	}
	client := action.NewGet(actionConfig)
	return client.Run(h.ReleaseName)
}

func (h *HelmHandler) IsDeployed() bool {
	rel, err := h.CheckStatus()
	if err != nil {
		return false
	}

	if rel != nil {
		if rel.Info.Status == release.StatusDeployed {
			return true
		}
	}
	return false
}

func (h *HelmHandler) installOCIChart() error {
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(h.settings.RESTClientGetter(), h.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
	})
	if err != nil {
		return fmt.Errorf("error initializing OCI chart install action: %s", err)
	}

	client := action.NewInstall(actionConfig)
	client.Namespace = h.Namespace
	client.ReleaseName = h.ReleaseName

	get, err := getter.NewOCIGetter()
	if err != nil {
		return fmt.Errorf("error creating a new OCI getter: %s", err)
	}

	b, err := get.Get(h.URL)
	if err != nil {
		return fmt.Errorf("error downloading the OCI chart %s : %s", h.URL, err)
	}

	tmpDir := os.TempDir()
	defer os.Remove(tmpDir)

	chartPath := filepath.Join(tmpDir, h.ChartName)
	err = os.WriteFile(chartPath, b.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("error saving the OCI chart: %s", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("error loading the OCI chart: %s", err)
	}

	p := getter.All(h.settings)
	valueOpts := &values.Options{}
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return err
	}
	if err := strvals.ParseInto(h.Args["set"], vals); err != nil {
		return errors.Wrap(err, "failed parsing --set data")
	}

	_, err = client.Run(chart, vals)
	if err != nil {
		return fmt.Errorf("error installing the OCI chart: %s", err)
	}
	return nil
}

func isOCIURL(url string) bool {
	u, err := neturl.Parse(url)
	if err != nil {
		return false
	}
	return u.Scheme == "oci"
}
