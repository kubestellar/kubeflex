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
		Run: func(cmd *cobra.Command, args []string) {
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			toSwitch, _ := cmd.Flags().GetBool(SwitchCtxFlag)
			cp := common.NewCP(kubeconfig, common.WithName(args[0]))
			ExecuteCtxRename(cp, args[1], toSwitch)
		},
	}
	command.Flags().BoolP(SwitchCtxFlag, "S", false, "switch to context after renaming process")
	return command
}

// Execute kflex ctx rename command
func ExecuteCtxRename(cp common.CP, newName string, toSwitch bool) error {
	kconf, err := kubeconfig.LoadKubeconfig(cp.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no kubeconfig context for %s was found, cannot load: %s\n", cp.Name, err)
		return err
	}
	newClusterName := certs.GenerateClusterName(newName)
	newAuthInfoAdminName := certs.GenerateAuthInfoAdminName(newName)
	newCtxName := certs.GenerateContextName(newName)
	kconf.Contexts[newCtxName] = &api.Context{
		Cluster:  newClusterName,
		AuthInfo: newAuthInfoAdminName,
	}
	kubeconfig.RenameKey(kconf.Clusters, certs.GenerateClusterName(cp.Name), newClusterName)
	kubeconfig.RenameKey(kconf.AuthInfos, certs.GenerateAuthInfoAdminName(cp.Name), newAuthInfoAdminName)
	fmt.Fprintf(os.Stdout, "renaming context from %s to %s\n", cp.Name, newCtxName)
	if err = kubeconfig.DeleteContext(kconf, cp.Name); err != nil {
		fmt.Fprintf(os.Stderr, "no kubeconfig context for %s was found: %s\n", cp.Name, err)
		return err
	}
	fmt.Fprintf(os.Stdout, "context %s is deleted\n", cp.Name)
	if toSwitch {
		fmt.Fprintf(os.Stdout, "switching context to %s\n", newCtxName)
		kconf.CurrentContext = newCtxName
	}
	if err = kubeconfig.WriteKubeconfig(cp.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		return err
	}
	return nil
}
