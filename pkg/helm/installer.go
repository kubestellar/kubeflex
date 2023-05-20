package helm

import (
	"context"
	"fmt"
	"log"
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

type HelmHandler struct {
	URL         string
	RepoName    string
	ChartName   string
	ReleaseName string
	Namespace   string
	Args        map[string]string
	log         logr.Logger
	settings    *cli.EnvSettings
}

func Init(ctx context.Context, handler *HelmHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	handler.log = clog.FromContext(ctx)
	handler.settings = cli.New()
	handler.settings.SetNamespace(handler.Namespace)
	return nil
}

func (h *HelmHandler) Install() error {
	// Add helm repo
	if err := h.repoAdd(); err != nil {
		return err
	}

	// Update charts from the helm repo
	if err := h.repoUpdate(); err != nil {
		return err
	}

	// Install charts
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
		//return fmt.Errorf("repo with name %s already exists", h.RepoName)
		h.log.Info("repo already exists", "repoName", h.RepoName)
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
	h.log.Info("repo has been added to your repositories", "name", h.RepoName)
	return nil
}

// RepoUpdate updates charts for all helm repos
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

	h.log.Info("getting the latest from chart repositories")
	var wg sync.WaitGroup
	for _, re := range repos {
		wg.Add(1)
		go func(re *repo.ChartRepository) {
			defer wg.Done()
			if _, err := re.DownloadIndexFile(); err != nil {
				h.log.Error(err, "...Unable to get an update from the chart repository", "name", re.Config.Name, "url", re.Config.URL)
			} else {
				h.log.Info("successfully got an update from the chart repository", "name", re.Config.Name)
			}
		}(re)
	}
	wg.Wait()
	h.log.Info("update complete!")
	return nil
}

// InstallChart
func (h *HelmHandler) chartInstall() error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(h.settings.RESTClientGetter(), h.settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
		return err
	}
	client := action.NewInstall(actionConfig)

	if client.Version == "" && client.Devel {
		client.Version = ">0.0.0-0"
	}
	//name, chart, err := client.NameAndChart(args)
	client.ReleaseName = h.ReleaseName
	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", h.RepoName, h.ChartName), h.settings)
	if err != nil {
		return err
	}

	h.log.Info("chart path", "path", cp)

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

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return err
	}

	validInstallableChart, err := isChartInstallable(chartRequested)
	if !validInstallableChart {
		return err
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
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
	h.log.Info(release.Manifest)
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
	if err != nil && rel != nil {
		if rel.Info.Status == release.StatusDeployed {
			return true
		}
	}
	return false
}
