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

package ctx

import (
	"fmt"
	"os"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	SwitchCtxFlag = "switch"
)

// Command kflex ctx rename
func CommandRename() *cobra.Command {
	command := &cobra.Command{
		Use:   "rename CONTEXT NEW_CONTEXT",
		Short: "Rename a context",
		Long:  `Rename a context in the kubeconfig file`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			toSwitch, _ := cmd.Flags().GetBool(SwitchCtxFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			return ExecuteCtxRename(cp, args[0], args[1], toSwitch)
		},
	}
	command.Flags().BoolP(SwitchCtxFlag, "S", false, "switch to context after renaming process")
	return command
}

// Execute kflex ctx rename command
func ExecuteCtxRename(cp common.CP, ctxName string, newCtxName string, toSwitch bool) error {
	kconf, err := kubeconfig.LoadKubeconfig(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	if _, ok := kconf.Contexts[ctxName]; !ok {
		return fmt.Errorf("no context '%s' is found in the kubeconfig", ctxName)
	}
	newClusterName := certs.GenerateClusterName(newCtxName)
	newAuthInfoAdminName := certs.GenerateAuthInfoAdminName(newCtxName)
	newCtxName = certs.GenerateContextName(newCtxName)
	kconf.Contexts[newCtxName] = &api.Context{
		Cluster:  newClusterName,
		AuthInfo: newAuthInfoAdminName,
	}
	if cluster, ok := kconf.Clusters[certs.GenerateClusterName(ctxName)]; ok {
		fmt.Fprintf(os.Stdout, "renaming cluster to %s\n", newClusterName)
		newCluster := *cluster
		kconf.Clusters[newClusterName] = &newCluster
	}
	if authInfo, ok := kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)]; ok {
		fmt.Fprintf(os.Stdout, "renaming user to %s\n", newAuthInfoAdminName)
		newAuthInfo := *authInfo
		kconf.AuthInfos[newAuthInfoAdminName] = &newAuthInfo
	}
	fmt.Printf("kconf-clusters: %v\n", kconf.Clusters)
	fmt.Printf("kconf-auths: %v\n", kconf.AuthInfos)
	fmt.Fprintf(os.Stdout, "renaming context from %s to %s\n", ctxName, newCtxName)
	if err = kubeconfig.DeleteAll(kconf, ctxName); err != nil {
		return fmt.Errorf("cannot delete context %s from kubeconfig: %v", ctxName, err)
	}
	fmt.Fprintf(os.Stdout, "context %s is deleted\n", ctxName)
	if toSwitch {
		fmt.Fprintf(os.Stdout, "switching context to %s\n", newCtxName)
		kconf.CurrentContext = newCtxName
	} else if kconf.CurrentContext == ctxName {
		fmt.Fprintf(os.Stdout, "switching to hosting cluster context\n")
		kubeconfig.SwitchToHostingClusterContext(kconf, false)
	}
	if err = kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	return nil
}
