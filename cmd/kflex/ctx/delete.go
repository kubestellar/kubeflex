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
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	// "sync"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
)

type DeleteOptions struct {
	Force bool
}

type DeleteOption func(*DeleteOptions)

// WithForce sets the force option to bypass confirmation
func WithForce() DeleteOption {
	return func(opts *DeleteOptions) {
		opts.Force = true
	}
}

func CommandDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete CONTEXT",
		Short: "Delete a context",
		Long:  `Delete a context in the kubeconfig file`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			chattyStatus, _ := cmd.Flags().GetBool(common.ChattyStatusFlag)
			force, _ := cmd.Flags().GetBool("force")
			cp := common.NewCP(kubeconfig)

			var options []DeleteOption
			if force {
				options = append(options, WithForce())
			}

			return ExecuteCtxDelete(cp, args[0], chattyStatus, options...)
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	return cmd
}

// Execute kflex ctx delete
func ExecuteCtxDelete(cp common.CP, ctxName string, chattyStatus bool, options ...DeleteOption) error {
	opts := &DeleteOptions{}
	for _, option := range options {
		option(opts)
	}

	var wg sync.WaitGroup
	done := make(chan bool)

	kconf, err := kubeconfig.LoadKubeconfig(cp.Kubeconfig)
	if err != nil {
		return fmt.Errorf("error loading kubeconfig: %v", err)
	}

	// Check if context is managed by KubeFlex and force is not set
	if !kubeconfig.IsContextManagedByKubeflex(kconf, ctxName) && !opts.Force {
		fmt.Printf("Warning: Context '%s' is not managed by KubeFlex.\n", ctxName)
		fmt.Print("Are you sure you want to delete this context? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading user input: %v", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	util.PrintStatus("Deleting context", done, &wg, chattyStatus)

	if err = kubeconfig.DeleteAll(kconf, ctxName); err != nil {
		return fmt.Errorf("error deleting context %s from kubeconfig: %v", ctxName, err)
	}
	if kconf.CurrentContext == ctxName {
		fmt.Printf("prepare the switch to hosting cluster context")
		kubeconfig.SwitchToHostingClusterContext(kconf)
	}
	if err = kubeconfig.WriteKubeconfig(cp.Kubeconfig, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	done <- true
	wg.Wait()
	return nil
}
