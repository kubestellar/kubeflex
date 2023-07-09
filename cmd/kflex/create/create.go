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

package create

import (
	"context"
	"fmt"
	"os"
	"sync"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CPCreate struct {
	common.CP
}

// Create a ne control plane
func (c *CPCreate) Create(controlPlaneType, backendType string) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.Context()

	cl := *(kfclient.GetClient(c.Kubeconfig))

	cp := c.generateControlPlane(controlPlaneType, backendType)

	util.PrintStatus(fmt.Sprintf("Creating new control plane %s...", c.Name), done, &wg)
	if err := cl.Create(context.TODO(), cp, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating instance: %v\n", err)
		os.Exit(1)
	}
	done <- true

	clientset := *(kfclient.GetClientSet(c.Kubeconfig))

	util.PrintStatus("Waiting for API server to become ready...", done, &wg)
	kubeconfig.WatchForSecretCreation(clientset, c.Name, util.GetKubeconfSecretNameByControlPlaneType(controlPlaneType))

	if controlPlaneType == string(tenancyv1alpha1.ControlPlaneTypeVCluster) {
		if err := util.WaitForStatefulSetReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(cp.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for stateful set to become ready: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := util.WaitForDeploymentReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(cp.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for deployment to become ready: %v\n", err)
			os.Exit(1)
		}
	}
	done <- true

	if err := kubeconfig.LoadAndMerge(c.Ctx, clientset, c.Name, controlPlaneType); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %v\n", err)
		os.Exit(1)
	}

	wg.Wait()
}

func (c *CPCreate) generateControlPlane(controlPlaneType, backendType string) *tenancyv1alpha1.ControlPlane {
	return &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.Name,
		},
		Spec: tenancyv1alpha1.ControlPlaneSpec{
			Type:    tenancyv1alpha1.ControlPlaneType(controlPlaneType),
			Backend: tenancyv1alpha1.BackendDBType(backendType),
		},
	}
}
