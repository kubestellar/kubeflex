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
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// defaults
const (
	BKTypeDefault        = string(tenancyv1alpha1.BackendDBTypeShared) // REFACTOR? local to create or common
	CTypeDefault         = string(tenancyv1alpha1.ControlPlaneTypeK8S) // REFACTOR? local to create or common
	ControlPlaneTypeFlag = "type"
	BackendTypeFlag      = "backend-type"
)

// REFACTOR: removed variables such as `hookVars` as they are used by multiple commands (create and adopt...). It should be defined locally to each command package instead of pointing to the same variable defined in main avoiding extreme edge cases.

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a control plane instance",
		Long: `Create a control plane instance and switches the Kubeconfig context to
	        the current instance`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			cpType, _ := flagset.GetString(ControlPlaneTypeFlag)
			backendType, _ := flagset.GetString(BackendTypeFlag)
			postCreateHook, _ := flagset.GetString(common.PostCreateHookFlag)
			hookVars, _ := flagset.GetStringArray(common.SetFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			// create passing the control plane type and backend type
			execute(cp, cpType, backendType, postCreateHook, hookVars, chattyStatus)
		},
	}

	flagset := command.Flags()
	// REFACTOR: putting CTypeDefault as default value, hence if empty string given to the flag, it picks up
	flagset.StringP(ControlPlaneTypeFlag, "t", CTypeDefault, "type of control plane: k8s|ocm|vcluster")
	// REFACTOR: same than CTypeDefault for BKTypeDefault
	flagset.StringP(BackendTypeFlag, "b", BKTypeDefault, "backend DB sharing: shared|dedicated")
	flagset.StringP(common.PostCreateHookFlag, "p", "", "name of post create hook to run")
	flagset.BoolP(common.ChattyStatusFlag, "s", true, "chatty status indicator")
	flagset.StringArrayP(common.SetFlag, "e", []string{}, "set post create hook variables, in the form name=value ")
	return command
}

// Create a new control plane
func execute(cp common.CP, controlPlaneType string, backendType string, hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.Context(chattyStatus, false, false, false)

	cl, err := kfclient.GetClient(cp.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting client: %v\n", err)
		os.Exit(1)
	}

	controlPlane, err := common.GenerateControlPlane(cp.Name, controlPlaneType, backendType, hook, hookVars, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating control plane object: %v\n", err)
		os.Exit(1)
	}

	util.PrintStatus(fmt.Sprintf("Creating new control plane %s of type %s ...", cp.Name, controlPlaneType), done, &wg, chattyStatus)
	if err := cl.Create(context.TODO(), controlPlane, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating instance: %v\n", err)
		os.Exit(1)
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(cp.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}
	clientset := *clientsetp

	util.PrintStatus("Waiting for API server to become ready...", done, &wg, chattyStatus)
	kubeconfig.WatchForSecretCreation(clientset, cp.Name, util.GetKubeconfSecretNameByControlPlaneType(controlPlaneType))

	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeHost):
		// hosting cluster is always ready
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		if err := util.WaitForStatefulSetReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(controlPlane.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for stateful set to become ready: %v\n", err)
			os.Exit(1)
		}
	case string(tenancyv1alpha1.ControlPlaneTypeK8S), string(tenancyv1alpha1.ControlPlaneTypeOCM):
		if err := util.WaitForDeploymentReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(controlPlane.Name)); err != nil {

			fmt.Fprintf(os.Stderr, "Error waiting for deployment to become ready: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown control plane type: %s\n", controlPlaneType)
		os.Exit(1)
	}

	done <- true

	if err := kubeconfig.LoadAndMerge(cp.Ctx, clientset, cp.Name, controlPlaneType); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %s\n", err)
		os.Exit(1)
	}

	wg.Wait()
}
