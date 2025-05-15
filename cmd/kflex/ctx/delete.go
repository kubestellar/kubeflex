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
	"sync"

	// "sync"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
)

func CommandDelete() *cobra.Command {
	return &cobra.Command{
		Use:   "delete CONTEXT",
		Short: "Delete a context",
		Long:  `Delete a context in the kubeconfig file`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			chattyStatus, _ := cmd.Flags().GetBool(common.ChattyStatusFlag)
			cp := common.NewCP(kubeconfig)
			return ExecuteCtxDelete(cp, args[0], chattyStatus)
		},
	}
}

// Execute kflex ctx delete
func ExecuteCtxDelete(cp common.CP, ctxName string, chattyStatus bool) error {
	var wg sync.WaitGroup
	done := make(chan bool)
	util.PrintStatus("Deleting context", done, &wg, chattyStatus)
	kconf, err := kubeconfig.LoadKubeconfig(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %v", err)
	}
	if err = kubeconfig.DeleteAll(kconf, ctxName); err != nil {
		return fmt.Errorf("error deleting context %s from kubeconfig: %v", ctxName, err)
	}
	if kconf.CurrentContext == ctxName {
		fmt.Printf("prepare the switch to hosting cluster context")
		kubeconfig.SwitchToHostingClusterContext(kconf, false)
	}
	if err = kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	done <- true
	wg.Wait()
	return nil
}
