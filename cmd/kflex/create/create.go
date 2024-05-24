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
	"strings"
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
func (c *CPCreate) Create(controlPlaneType, backendType, hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.Context(chattyStatus, false)

	clp, err := kfclient.GetClient(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting client: %v\n", err)
		os.Exit(1)
	}
	cl := *clp

	cp := c.generateControlPlane(controlPlaneType, backendType, hook, hookVars)

	util.PrintStatus(fmt.Sprintf("Creating new control plane %s of type %s ...", c.Name, controlPlaneType), done, &wg, chattyStatus)
	if err := cl.Create(context.TODO(), cp, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating instance: %v\n", err)
		os.Exit(1)
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp

	util.PrintStatus("Waiting for API server to become ready...", done, &wg, chattyStatus)
	kubeconfig.WatchForSecretCreation(clientset, c.Name, util.GetKubeconfSecretNameByControlPlaneType(controlPlaneType))

	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeHost):
		// hosting cluster is always ready
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		if err := util.WaitForStatefulSetReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(cp.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for stateful set to become ready: %v\n", err)
			os.Exit(1)
		}
	case string(tenancyv1alpha1.ControlPlaneTypeK8S), string(tenancyv1alpha1.ControlPlaneTypeOCM):
		if err := util.WaitForDeploymentReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(cp.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for deployment to become ready: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown control plane type: %s\n", controlPlaneType)
		os.Exit(1)
	}

	done <- true

	if err := kubeconfig.LoadAndMerge(c.Ctx, clientset, c.Name, controlPlaneType); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %s\n", err)
		os.Exit(1)
	}

	wg.Wait()
}

func (c *CPCreate) generateControlPlane(controlPlaneType, backendType, hook string, hookVars []string) *tenancyv1alpha1.ControlPlane {
	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.Name,
		},
		Spec: tenancyv1alpha1.ControlPlaneSpec{
			Type:    tenancyv1alpha1.ControlPlaneType(controlPlaneType),
			Backend: tenancyv1alpha1.BackendDBType(backendType),
		},
	}
	if hook != "" {
		cp.Spec.PostCreateHook = &hook
		cp.Spec.PostCreateHookVars = convertToMap(hookVars)
	}
	return cp
}

func convertToMap(pairs []string) map[string]string {
	params := make(map[string]string)

	for _, pair := range pairs {
		split := strings.SplitN(pair, "=", 2)
		if len(split) == 2 {
			params[split[0]] = split[1]
		}
	}

	return params
}
