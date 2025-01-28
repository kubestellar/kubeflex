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

package delete

import (
	"context"
	"fmt"
	"os"
	"sync"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CPDelete struct {
	common.CP
}

func (c *CPDelete) Delete(chattyStatus bool) {
	done := make(chan bool)
	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Name,
		},
	}
	var wg sync.WaitGroup

	util.PrintStatus(fmt.Sprintf("Deleting control plane %s...", c.Name), done, &wg, chattyStatus)
	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	if err = kubeconfig.SwitchToHostingClusterContext(kconf, false); err != nil {
		fmt.Fprintf(os.Stderr, "error switching to hosting cluster kubeconfig context: %s\n", err)
		os.Exit(1)
	}

	if err := kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}

	kfcClient, err := kfclient.GetClient(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kf client: %s\n", err)
		os.Exit(1)
	}

	if err := kfcClient.Get(context.TODO(), client.ObjectKeyFromObject(cp), cp, &client.GetOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "control plane not found on server: %s", err)
	}

	if err := kfcClient.Delete(context.TODO(), cp, &client.DeleteOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting instance: %s\n", err)
		os.Exit(1)
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kf client: %s\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp
	util.PrintStatus(fmt.Sprintf("Waiting for control plane %s to be deleted...", c.Name), done, &wg, chattyStatus)
	util.WaitForNamespaceDeletion(clientset, util.GenerateNamespaceFromControlPlaneName(c.Name))

	if cp.Spec.Type != tenancyv1alpha1.ControlPlaneTypeHost &&
		cp.Spec.Type != tenancyv1alpha1.ControlPlaneTypeExternal {
		if err := kubeconfig.DeleteContext(kconf, c.Name); err != nil {
			fmt.Fprintf(os.Stderr, "no kubeconfig context for %s was found: %s\n", c.Name, err)
			os.Exit(1)
		}

		if err := kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
			os.Exit(1)
		}
	}

	done <- true
	wg.Wait()
}
