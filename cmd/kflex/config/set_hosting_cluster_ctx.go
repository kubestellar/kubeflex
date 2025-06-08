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

package config

import (
	"fmt"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

func CommandSetHostingClusterCtx() *cobra.Command {
	command := &cobra.Command{
		Use:   "set-hosting",
		Short: "Set hosting cluster context",
		Long:  `Set hosting cluster context name of kubeflex within kubeconfig file`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagset := cmd.Flags()
			kubeconfigFile, err := flagset.GetString(common.KubeconfigFlag)
			if err != nil {
				return fmt.Errorf("error while parsing --kubeconfig: %v", err)
			}
			return ExecuteSetHostingClusterCtx(kubeconfigFile, args[0])
		},
	}
	return command
}

// Set hosting cluster context name
func ExecuteSetHostingClusterCtx(kubeconfigFile string, ctxName string) error {
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigFile)
	if err != nil {
		return fmt.Errorf("error while executing set hosting cluster ctx: %v", err)
	}
	fmt.Printf("setting hosting cluster context name to '%s' in %s", ctxName, kubeconfigFile)
	err = kubeconfig.SetHostingClusterContext(kconf, &ctxName)
	if err != nil {
		return fmt.Errorf("error while executing set hosting cluster ctx: %v", err)
	}
	err = kubeconfig.WriteKubeconfig(kubeconfigFile, kconf)
	if err != nil {
		return fmt.Errorf("error while executing set hosting cluster ctx: %v", err)
	}
	return nil
}
