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

package init

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/cmd/kflex/init/cluster"
	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/helm"
	kcfg "github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
)

const (
	CreateKindFlag        = "create-kind"
	DomainFlag            = "domain"
	HostContainerNameFlag = "host-container-name" // REFACTOR? replace with host-container-name?
	ExternalPortFlag      = "external-port"       // REFACTOR? replace with external-port?
	DefaultKindClusterName = "kind-kubeflex"       // Default cluster name for kind clusters
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "init [cluster-name]",
		Short: "Initialize kubeflex",
		Long:  `Installs the default shared storage backend and the kubeflex operator`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			createkind, _ := flagset.GetBool(CreateKindFlag)
			domain, _ := flagset.GetString(DomainFlag)
			externalPort, _ := flagset.GetInt(ExternalPortFlag)
			hostContainer, _ := flagset.GetString(HostContainerNameFlag)
			
			// Handle positional cluster name parameter
			clusterName := DefaultKindClusterName // default
			if len(args) > 0 {
				clusterName = args[0]
			}
			done := make(chan bool)
			var wg sync.WaitGroup
			var isOCP bool

			util.PrintStatus("Checking if OpenShift cluster...", done, &wg, chattyStatus)
			clientsetp, err := client.GetClientSet(kubeconfig)
			if err == nil {
				isOCP = util.IsOpenShift(*clientsetp)
				if isOCP {
					done <- true
					util.PrintStatus("OpenShift cluster detected", done, &wg, chattyStatus)
				}
			}
			done <- true

			if createkind {
				if isOCP {
					return fmt.Errorf("openShift cluster detected on existing context\nSwitch to a non-OpenShift context with `kubectl config use-context <context-name>` and retry")
				}
				cluster.CreateKindCluster(chattyStatus, clusterName)
			}

			cp := common.NewCP(kubeconfig)
			err = ExecuteInit(cp.Ctx, cp.Kubeconfig, common.Version, common.BuildDate, domain, strconv.Itoa(externalPort), hostContainer, chattyStatus, isOCP)
			wg.Wait()
			return err
		},
	}
	flagset := command.Flags()
	flagset.BoolP(CreateKindFlag, "c", false, "Create and configure a kind cluster for installing Kubeflex")
	flagset.StringP(DomainFlag, "d", "localtest.me", "domain for FQDN")
	flagset.StringP(HostContainerNameFlag, "n", "kubeflex-control-plane", "Name of the hosting cluster container (kind or k3d only)")
	flagset.IntP(ExternalPortFlag, "p", 9443, "external port used by ingress")
	return command
}

func ExecuteInit(ctx context.Context, kubeconfig, version, buildDate string, domain, externalPort, hostContainer string, chattyStatus, isOCP bool) error {
	done := make(chan bool)
	var wg sync.WaitGroup

	clientsetp, err := client.GetClientSet(kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}
	clientset := *clientsetp

	util.PrintStatus(fmt.Sprintf("Kubeflex %s %s", version, buildDate), done, &wg, chattyStatus)
	done <- true

	util.PrintStatus("Setting hosting cluster preference in kubeconfig", done, &wg, chattyStatus)

	kconfig, err := kcfg.LoadKubeconfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %v", err)
	}
	err = kcfg.SetHostingClusterContext(kconfig, nil)
	if err != nil {
		return fmt.Errorf("error setting hosting cluster context: %v", err)
	}
	err = kcfg.WriteKubeconfig(kubeconfig, kconfig)
	if err != nil {
		return fmt.Errorf("error writing hosting cluster in kubeconfig: %v", err)
	}
	done <- true

	util.PrintStatus("Ensuring kubeflex-system namespace...", done, &wg, chattyStatus)
	err = ensureSystemNamespace(kubeconfig, util.SystemNamespace)
	if err != nil {
		return err
	}
	done <- true

	util.PrintStatus("Installing shared backend DB...", done, &wg, chattyStatus)
	err = ensureSystemDB(ctx, isOCP)
	if err != nil {
		return err
	}
	done <- true

	util.PrintStatus("Waiting for shared backend DB to become ready...", done, &wg, chattyStatus)
	util.WaitForStatefulSetReady(
		clientset,
		util.GeneratePSReplicaSetName(util.DBReleaseName),
		util.SystemNamespace)
	done <- true

	util.PrintStatus("Installing kubeflex operator...", done, &wg, chattyStatus)
	err = ensureKFlexOperator(ctx, version, domain, externalPort, hostContainer, isOCP)
	if err != nil {
		return err
	}
	done <- true

	util.PrintStatus("Waiting for kubeflex operator to become ready...", done, &wg, chattyStatus)
	util.WaitForDeploymentReady(
		clientset,
		util.GenerateOperatorDeploymentName(),
		util.SystemNamespace)
	done <- true

	wg.Wait()
	return nil
}

func ensureSystemDB(ctx context.Context, isOCP bool) error {
	vars := []string{
		"primary.extendedConfiguration=max_connections=1000",
		"primary.priorityClassName=system-node-critical",
	}
	if isOCP {
		vars = append(vars,
			"primary.podSecurityContext.fsGroup=null",
			"primary.podSecurityContext.seccompProfile.type=RuntimeDefault",
			"primary.containerSecurityContext.runAsUser=null",
			"primary.containerSecurityContext.allowPrivilegeEscalation=false",
			"primary.containerSecurityContext.runAsNonRoot=true",
			"primary.containerSecurityContext.seccompProfile.type=RuntimeDefault",
			"volumePermissions.enabled=false",
			"shmVolume.enabled=false",
		)
	}
	h := &helm.HelmHandler{
		URL:         PostgresURL,
		RepoName:    PostgresRepoName,
		ChartName:   PostgresChartName,
		Namespace:   util.SystemNamespace,
		ReleaseName: PostgresReleaseName,
		Args:        map[string]string{"set": strings.Join(vars, ",")},
	}
	err := helm.Init(ctx, h)
	if err != nil {
		return fmt.Errorf("error initializing helm: %v", err)
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			return fmt.Errorf("error installing chart: %v", err)
		}
	}
	return nil
}

func ensureKFlexOperator(ctx context.Context, fullVersion, domain, externalPort, hostContainer string, isOCP bool) error {
	version := util.ParseVersionNumber(fullVersion)
	vars := []string{
		fmt.Sprintf("version=%s", version),
		fmt.Sprintf("domain=%s", domain),
		fmt.Sprintf("externalPort=%s", externalPort),
		fmt.Sprintf("hostContainer=%s", hostContainer),
		fmt.Sprintf("isOpenShift=%s", strconv.FormatBool(isOCP)),
		"installPostgreSQL=false",
	}
	h := &helm.HelmHandler{
		URL:         fmt.Sprintf("%s:%s", KflexOperatorURL, version),
		RepoName:    KflexOperatorRepoName,
		ChartName:   KflexOperatorChartName,
		Namespace:   util.SystemNamespace,
		ReleaseName: KflexOperatorReleaseName,
		Args:        map[string]string{"set": strings.Join(vars, ",")},
	}
	err := helm.Init(ctx, h)
	if err != nil {
		return fmt.Errorf("error initializing helm: %v", err)
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			return fmt.Errorf("error installing chart: %v", err)
		}
	}
	return nil
}

func ensureSystemNamespace(kubeconfig, namespace string) error {
	clientsetp, err := client.GetClientSet(kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
	}
	clientset := *clientsetp
	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("error creating system namespace: %v", err)
			}
		}
	}
	return nil
}
