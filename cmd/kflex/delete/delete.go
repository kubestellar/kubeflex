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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CPDelete struct {
	common.CP
}

func (c *CPDelete) Delete() {
	done := make(chan bool)
	cp := c.generateControlPlane()
	var wg sync.WaitGroup

	util.PrintStatus(fmt.Sprintf("Deleting control plane %s...", c.Name), done, &wg)
	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	if err = kubeconfig.SwitchToInitialContext(kconf, false); err != nil {
		fmt.Fprintf(os.Stderr, "no initial kubeconfig context was found: %s\n", err)
	}

	if err = kubeconfig.DeleteContext(kconf, c.Name); err != nil {
		fmt.Fprintf(os.Stderr, "no kubeconfig context for %s was found: %s\n", c.Name, err)
	}

	if err = kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}

	kfcClient := *(kfclient.GetClient(c.Kubeconfig))
	if err := kfcClient.Delete(context.TODO(), cp, &client.DeleteOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting instance: %s\n", err)
		os.Exit(1)
	}
	done <- true

	clientset := *(kfclient.GetClientSet(c.Kubeconfig))
	util.PrintStatus(fmt.Sprintf("Waiting for control plane %s to be deleted...", c.Name), done, &wg)
	util.WaitForNamespaceDeletion(clientset, util.GenerateNamespaceFromControlPlaneName(c.Name))

	done <- true
	wg.Wait()
}

func (c *CPDelete) generateControlPlane() *tenancyv1alpha1.ControlPlane {
	return &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.Name,
		},
	}
}
