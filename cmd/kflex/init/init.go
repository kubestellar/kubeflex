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
	"os"
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
	HostContainerNameFlag = "hostContainerName" // REFACTOR? replace with host-container-name?
	ExternalPortFlag      = "externalPort"      // REFACTOR? replace with external-port?
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "init",
		Short: "Initialize kubeflex",
		Long:  `Installs the default shared storage backend and the kubeflex operator`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			createkind, _ := flagset.GetBool(CreateKindFlag)
			domain, _ := flagset.GetString(DomainFlag)
			externalPort, _ := flagset.GetInt(ExternalPortFlag)
			hostContainer, _ := flagset.GetString(HostContainerNameFlag)
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
					fmt.Fprintf(os.Stderr, "OpenShift cluster detected on existing context\n")
					fmt.Fprintf(os.Stdout, "Switch to a non-OpenShift context with `kubectl config use-context <context-name>` and retry.\n")
					os.Exit(1)
				}
				cluster.CreateKindCluster(chattyStatus)
			}

			// REFACTOR: leverage CP struct to give Context and Kubeconfig
			cp := common.NewCP(kubeconfig)
			ExecuteInit(cp.Ctx, cp.Kubeconfig, common.Version, common.BuildDate, domain, strconv.Itoa(externalPort), hostContainer, chattyStatus, isOCP)
			wg.Wait()
		},
	}
	flagset := command.Flags()
	flagset.BoolP(CreateKindFlag, "c", false, "Create and configure a kind cluster for installing Kubeflex")
	flagset.StringP(DomainFlag, "d", "localtest.me", "domain for FQDN")
	flagset.StringP(HostContainerNameFlag, "n", "kubeflex-control-plane", "Name of the hosting cluster container (kind or k3d only)")
	flagset.IntP(ExternalPortFlag, "p", 9443, "external port used by ingress")
	return command
}

func ExecuteInit(ctx context.Context, kubeconfig, version, buildDate string, domain, externalPort, hostContainer string, chattyStatus, isOCP bool) {
	done := make(chan bool)
	var wg sync.WaitGroup

	clientsetp, err := client.GetClientSet(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp

	util.PrintStatus(fmt.Sprintf("Kubeflex %s %s", version, buildDate), done, &wg, chattyStatus)
	done <- true

	util.PrintStatus("Setting hosting cluster preference in kubeconfig", done, &wg, chattyStatus)
	err = kcfg.SaveHostingClusterContextPreference(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting hosting cluster context preference: %v\n", err)
		os.Exit(1)
	}
	done <- true

	util.PrintStatus("Ensuring kubeflex-system namespace...", done, &wg, chattyStatus)
	ensureSystemNamespace(kubeconfig, util.SystemNamespace)
	done <- true

	util.PrintStatus("Installing shared backend DB...", done, &wg, chattyStatus)
	ensureSystemDB(ctx, isOCP)
	done <- true

	util.PrintStatus("Waiting for shared backend DB to become ready...", done, &wg, chattyStatus)
	util.WaitForStatefulSetReady(
		clientset,
		util.GeneratePSReplicaSetName(util.DBReleaseName),
		util.SystemNamespace)
	done <- true

	util.PrintStatus("Installing kubeflex operator...", done, &wg, chattyStatus)
	ensureKFlexOperator(ctx, version, domain, externalPort, hostContainer, isOCP)
	done <- true

	util.PrintStatus("Waiting for kubeflex operator to become ready...", done, &wg, chattyStatus)
	util.WaitForDeploymentReady(
		clientset,
		util.GenerateOperatorDeploymentName(),
		util.SystemNamespace)
	done <- true

	wg.Wait()
}

func ensureSystemDB(ctx context.Context, isOCP bool) {
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
		fmt.Fprintf(os.Stderr, "Error initializing helm: %v\n", err)
		os.Exit(1)
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing chart: %v\n", err)
			os.Exit(1)
		}
	}
}

func ensureKFlexOperator(ctx context.Context, fullVersion, domain, externalPort, hostContainer string, isOCP bool) {
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
		fmt.Fprintf(os.Stderr, "Error initializing helm: %v\n", err)
		os.Exit(1)
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing chart: %v\n", err)
			os.Exit(1)
		}
	}
}

func ensureSystemNamespace(kubeconfig, namespace string) {
	clientsetp, err := client.GetClientSet(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
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
				fmt.Fprintf(os.Stderr, "Error creating system namespace: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
