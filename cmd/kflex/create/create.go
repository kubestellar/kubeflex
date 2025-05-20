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
        Run: func(cmd *cobra.Command, args []string) {
            flagset := cmd.Flags()
            kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
            chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
            cpType, _ := flagset.GetString(ControlPlaneTypeFlag)
            backendType, _ := flagset.GetString(BackendTypeFlag)
			postCreateHook, _ := flagset.GetString(common.PostCreateHookFlag)
			postCreateHooks, _ := flagset.GetStringArray("post-create-hook")
			hookVars, _ := flagset.GetStringArray(common.SetFlag)
            cp := common.NewCP(kubeconfig, common.WithName(args[0]))
            
            // Convert legacy parameters to new format
			hooks := parseHooks(postCreateHooks, postCreateHook, hookVars)
			
			// For backward compatibility, use the legacy single hook and vars
			var hookName string
			var hookVarsForLegacy []string
			if len(hooks) > 0 && hooks[0].HookName != nil {
				hookName = *hooks[0].HookName
				for k, v := range hooks[0].Vars {
					hookVarsForLegacy = append(hookVarsForLegacy, fmt.Sprintf("%s=%s", k, v))
				}
			}
			ExecuteCreate(cp, cpType, backendType, hookName, hookVarsForLegacy, chattyStatus)
        },
    }

    flagset := command.Flags()
    flagset.StringP(ControlPlaneTypeFlag, "t", CTypeDefault, "type of control plane: k8s|ocm|vcluster")
    flagset.StringP(BackendTypeFlag, "b", BKTypeDefault, "backend DB sharing: shared|dedicated")
    flagset.StringP(common.PostCreateHookFlag, "p", "", "name of post create hook to run (deprecated)")
	flagset.StringArrayP("post-create-hook", "hook", []string{}, "names of post create hooks to run")
    flagset.BoolP(common.ChattyStatusFlag, "s", true, "chatty status indicator")
    flagset.StringArrayP(common.SetFlag, "e", []string{}, "set post create hook variables (format: hookName.key=value)")
    return command
}

func parseHooks(newHooks []string, legacyHook string, vars []string) []tenancyv1alpha1.PostCreateHookUse {
    var hooks []tenancyv1alpha1.PostCreateHookUse
    
    // Handle legacy hook
    if legacyHook != "" {
        varsMap := parseVars(vars)
        hooks = append(hooks, tenancyv1alpha1.PostCreateHookUse{
            HookName: &legacyHook,
            Vars:     varsMap,
        })
    }
    
    // Handle new hooks
    for _, name := range newHooks {
        hooks = append(hooks, tenancyv1alpha1.PostCreateHookUse{
            HookName: &name,
            Vars:     parseVarsForHook(name, vars),
        })
    }
    
    return hooks
}

func parseVarsForHook(hookName string, vars []string) map[string]string {
	result := make(map[string]string)
	for _, pair := range vars {
		parts := strings.SplitN(pair, ".", 2)
		if len(parts) == 2 && parts[0] == hookName {
			keyVal := strings.SplitN(parts[1], "=", 2)
			if len(keyVal) == 2 {
				result[keyVal[0]] = keyVal[1]
			}
		}
	}
	return result
}

// parseVars parses legacy hook variables in the format key=value.
func parseVars(vars []string) map[string]string {
	result := make(map[string]string)
	for _, pair := range vars {
		keyVal := strings.SplitN(pair, "=", 2)
		if len(keyVal) == 2 {
			result[keyVal[0]] = keyVal[1]
		}
	}
	return result
}

// Create a new control plane
func ExecuteCreate(cp common.CP, controlPlaneType string, backendType string, hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.ExecuteCtx(chattyStatus, false, false, false)

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
