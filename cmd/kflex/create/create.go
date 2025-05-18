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
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	"github.com/kubestellar/kubeflex/pkg/certs"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// defaults
const (
	BKTypeDefault        = string(tenancyv1alpha1.BackendDBTypeShared)
	CTypeDefault         = string(tenancyv1alpha1.ControlPlaneTypeK8S)
	ControlPlaneTypeFlag = "type"
	BackendTypeFlag      = "backend-type"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a control plane instance",
		Long: `Create a control plane instance and switches the Kubeconfig context to
	        the current instance`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			cpType, _ := flagset.GetString(ControlPlaneTypeFlag)
			backendType, _ := flagset.GetString(BackendTypeFlag)
			postCreateHooks, _ := flagset.GetStringArray(common.PostCreateHookFlag)
			hookVars, _ := flagset.GetStringArray(common.SetFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			
			hooks := parseHooksAndVars(postCreateHooks, hookVars)
			ExecuteCreate(cp, cpType, backendType, hooks, chattyStatus)
		},
	}

	flagset := command.Flags()
	flagset.StringP(ControlPlaneTypeFlag, "t", CTypeDefault, "type of control plane: k8s|ocm|vcluster")
	flagset.StringP(BackendTypeFlag, "b", BKTypeDefault, "backend DB sharing: shared|dedicated")
	flagset.StringArrayP(common.PostCreateHookFlag, "p", []string{}, "name of post create hook to run")
	flagset.BoolP(common.ChattyStatusFlag, "s", true, "chatty status indicator")
	flagset.StringArrayP(common.SetFlag, "e", []string{}, "set post create hook variables in format hookName.key=value")
	return command
}

func parseHooksAndVars(hookNames []string, varArgs []string) []tenancyv1alpha1.PostCreateHookUse {
	varsByHook := make(map[string]map[string]string)
	
	for _, arg := range varArgs {
		parts := strings.SplitN(arg, ".", 2)
		if len(parts) != 2 {
			continue // handle error
		}
		hookName := parts[0]
		keyVal := strings.SplitN(parts[1], "=", 2)
		if len(keyVal) != 2 {
			continue // handle error
		}
		if varsByHook[hookName] == nil {
			varsByHook[hookName] = make(map[string]string)
		}
		varsByHook[hookName][keyVal[0]] = keyVal[1]
	}
	
	var hooks []tenancyv1alpha1.PostCreateHookUse
	for _, name := range hookNames {
		vars := varsByHook[name]
		hooks = append(hooks, tenancyv1alpha1.PostCreateHookUse{
			HookName: &name,
			Vars:     vars,
		})
	}
	return hooks
}

// Create a new control plane
func ExecuteCreate(cp common.CP, controlPlaneType string, backendType string, hooks []tenancyv1alpha1.PostCreateHookUse, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}                               // this is always undefined control plane, hence
	cx.ExecuteCtx(chattyStatus, false, false, false) // TODO replace by switch to hosting cluster

	cl, err := kfclient.GetClient(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting client: %v", err)
	}

	// Pass hooks as required by the new function signature
	controlPlane, err := common.GenerateControlPlane(cp.Name, controlPlaneType, backendType, hooks, nil)
	if err != nil {
		return fmt.Errorf("error generating control plane object: %v", err)
	}

	util.PrintStatus(fmt.Sprintf("Creating new control plane %s of type %s ...", cp.Name, controlPlaneType), done, &wg, chattyStatus)
	if err := cl.Create(context.TODO(), controlPlane, &client.CreateOptions{}); err != nil {
		return fmt.Errorf("error creating instance: %v", err)
	}
	done <- true

	clientsetp, err := kfclient.GetClientSet(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error getting clientset: %v", err)
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

			return fmt.Errorf("error waiting for stateful set to become ready: %v", err)
		}
	case string(tenancyv1alpha1.ControlPlaneTypeK8S), string(tenancyv1alpha1.ControlPlaneTypeOCM):
		if err := util.WaitForDeploymentReady(clientset,
			util.GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType),
			util.GenerateNamespaceFromControlPlaneName(controlPlane.Name)); err != nil {

			return fmt.Errorf("error waiting for deployment to become ready: %v", err)
		}
	default:
		return fmt.Errorf("unknown control plane type: %s", controlPlaneType)
	}

	done <- true
	kconf, err := kubeconfig.LoadAndMergeClientServerKubeconfig(cp.Ctx, cp.Kubeconfig, clientset, cp.Name, controlPlaneType)
	if err != nil {
		return fmt.Errorf("error loading and merging kubeconfig: %v", err)
	}
	if err = kubeconfig.AssignControlPlaneToContext(kconf, cp.Name, certs.GenerateContextName(cp.Name)); err != nil {
		return fmt.Errorf("error assigning control plane to context as kubeconfig extension: %v", err)
	}
	kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf)
	wg.Wait()
	return nil
}
