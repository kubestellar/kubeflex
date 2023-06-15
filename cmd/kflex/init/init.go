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
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/helm"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func Init(ctx context.Context, kubeconfig string) {
	done := make(chan bool)
	var wg sync.WaitGroup

	util.PrintStatus("Installing shared backend DB...", done, &wg)

	ensureSystemNamespace(kubeconfig, util.SystemNamespace)

	ensureSystemDB(ctx)
	done <- true

	util.PrintStatus("Waiting for shared backend DB to become ready...", done, &wg)
	util.WaitForStatefulSetReady(
		*(client.GetClientSet(kubeconfig)),
		util.GeneratePSReplicaSetName(util.DBReleaseName),
		util.SystemNamespace)
	done <- true

	util.PrintStatus("Installing kubeflex operator...", done, &wg)
	ensureKFlexOperator(ctx)
	done <- true

	wg.Wait()
}

func ensureSystemDB(ctx context.Context) {
	h := &helm.HelmHandler{
		URL:         PostgresURL,
		RepoName:    PostgresRepoName,
		ChartName:   PostgresChartName,
		Namespace:   util.SystemNamespace,
		ReleaseName: PostgresReleaseName,
		Args:        PostgresArgs,
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

func ensureKFlexOperator(ctx context.Context) {
	h := &helm.HelmHandler{
		URL:         KflexOperatorURL,
		RepoName:    KflexOperatorRepoName,
		ChartName:   KflexOperatorChartName,
		Namespace:   util.SystemNamespace,
		ReleaseName: KflexOperatorReleaseName,
		Args:        KflexOperatorArgs,
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
