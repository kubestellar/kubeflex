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

	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/helm"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func Init(ctx context.Context, kubeconfig, version, buildDate string, domain, externalPort string) {
	done := make(chan bool)
	var wg sync.WaitGroup

	util.PrintStatus(fmt.Sprintf("Kubeflex %s %s", version, buildDate), done, &wg)
	done <- true

	util.PrintStatus("Ensuring kubeflex-system namespace...", done, &wg)
	ensureSystemNamespace(kubeconfig, util.SystemNamespace)
	done <- true
	util.PrintStatus("Checking if OpenShift cluster...", done, &wg)
	isOCP := util.IsOpenShift(*(client.GetClientSet(kubeconfig)))
	if isOCP {
		done <- true
		util.PrintStatus("OpenShift cluster detected", done, &wg)
	}
	done <- true

	util.PrintStatus("Installing kubeflex operator...", done, &wg)
	ensureKFlexOperator(ctx, version, domain, externalPort, isOCP)
	done <- true

	if isOCP {
		util.PrintStatus("Adding OpenShift anyuid SCC to kubeflex SA...", done, &wg)
		util.AddSCCtoUserPolicy("")
		done <- true
	}

	util.PrintStatus("Waiting for kubeflex operator to become ready...", done, &wg)
	util.WaitForDeploymentReady(
		*(client.GetClientSet(kubeconfig)),
		util.GenerateOperatorDeploymentName(),
		util.SystemNamespace)
	done <- true

	wg.Wait()
}

func ensureKFlexOperator(ctx context.Context, fullVersion, domain, externalPort string, isOCP bool) {
	version := util.ParseVersionNumber(fullVersion)
	vars := []string{
		fmt.Sprintf("version=%s", version),
		fmt.Sprintf("domain=%s", domain),
		fmt.Sprintf("externalPort=%s", externalPort),
		fmt.Sprintf("isOpenShift=%s", strconv.FormatBool(isOCP)),
		"postgresql.primary.extendedConfiguration=max_connections=1000",
		"postgresql.primary.priorityClassName=system-node-critical",
	}
	if isOCP {
		vars = append(vars,
			"postgresql.primary.podSecurityContext.fsGroup=null",
			"postgresql.primary.podSecurityContext.seccompProfile.type=RuntimeDefault",
			"postgresql.primary.containerSecurityContext.runAsUser=null",
			"postgresql.primary.containerSecurityContext.allowPrivilegeEscalation=false",
			"postgresql.primary.containerSecurityContext.runAsNonRoot=true",
			"postgresql.primary.containerSecurityContext.seccompProfile.type=RuntimeDefault",
			"postgresql.volumePermissions.enabled=false",
			"postgresql.shmVolume.enabled=false",
		)
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
	client := client.GetClientSet(kubeconfig)

	_, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = client.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating system namespace: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
