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

package main

import (
	"fmt"
	"os"

	"github.com/kubestellar/kubeflex/cmd/kflex/adopt"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/cmd/kflex/create"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	"github.com/kubestellar/kubeflex/cmd/kflex/delete"
	kflexInit "github.com/kubestellar/kubeflex/cmd/kflex/init"
	"github.com/kubestellar/kubeflex/cmd/kflex/list"
	"github.com/kubestellar/kubeflex/cmd/kflex/version"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// Version injected by makefile:LDFLAGS
var Version string

// BuildDate injected by makefile:LDFLAGS
var BuildDate string

// REFACTOR: All global variables will disappear as each command package defines their own flags and retrieve then within cobra.Run function
var kubeconfig string // REFACTOR: to delete

var Hook string                 // REFACTOR: to delete
var chattyStatus bool           // REFACTOR: to delete
var overwriteExistingCtx bool   // REFACTOR: to delete
var setCurrentCtxAsHosting bool // REFACTOR: to delete

var rootCmd = &cobra.Command{
	Use:   "kflex",
	Short: "CLI for kubeflex",
	Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
}

// REFACTOR: all commands of kflex (non root) are moving into their own command package
// REFACTOR: to move to its own package (see how create command is implemented)

// REFACTOR: to move to its own package (see how create command is implemented)

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPAdopt as common.CP is enough

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPDelete as common.CP is enough

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPCtx as common.CP is enough
var ctxGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current kubeconfig context",
	Long:  `Retrieve and display the current context from the kubeconfig file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cp := cont.CPCtx{
			CP: common.NewCP(kubeconfig),
		}
		cp.GetCurrentContext()
	},
}

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPCtx as common.CP is enough
var ctxCmd = &cobra.Command{
	Use:   "ctx",
	Short: "Switch or get kubeconfig context",
	Long: `Running without an argument switches the context back to the hosting cluster context,
			        while providing the control plane name as argument switches the context to
					that control plane. Use 'get' to retrieve the current context.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		aliasName, _ := cmd.Flags().GetString("alias")
		cpName := ""
		if len(args) == 1 {
			cpName = args[0]
		}
		cp := cont.CPCtx{
			CP: common.NewCP(kubeconfig, common.WithName(cpName), common.WithAliasName(aliasName)),
		}
		cp.Context(chattyStatus, true, overwriteExistingCtx, setCurrentCtxAsHosting)
	},
}

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPCtx as common.CP is enough
var listCtxCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available contexts",
	Long:  `List all available contexts in the kubeconfig file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cp := cont.CPCtx{
			CP: common.NewCP(kubeconfig),
		}
		cp.ListContexts()
	},
}

func init() {
	// REFACTOR: init() will only define PersistentFlag and add Commands
	// REFACTOR: PersistentFlags makes flag across commands and subcommands which is the behaviour expected prior refactoring
	pflagset := rootCmd.PersistentFlags()
	pflagset.StringP(common.KubeconfigFlag, "k", clientcmd.RecommendedFileName, "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, and $KUBECONFIG is set, it uses the value in $KUBECONFIG, otherwise it falls back to ${HOME}/.kube/config")
	pflagset.BoolP(common.ChattyStatusFlag, "s", true, "chatty status indicator")
	pflagset.IntP(common.VerbosityFlag, "v", 0, "log level") // TODO - figure out how to inject verbosity

	// REFACTOR: to move to its own package (see how create command is implemented)

	// REFACTOR: to move to its own package (see how create command is implemented)

	// REFACTOR: to move to its own package (see how create command is implemented)
	ctxCmd.Flags().BoolVarP(&overwriteExistingCtx, "overwrite-existing-context", "o", false, "Overwrite of hosting cluster context with new control plane context")
	ctxCmd.Flags().BoolVarP(&setCurrentCtxAsHosting, "set-current-for-hosting", "c", false, "Set current context as hosting cluster context")
	ctxCmd.Flags().String("alias", "", "Set an alias name as the context, user and cluster value instead of cp name")
	// REFACTOR: to move to its own package (see how create command is implemented)
	ctxCmd.AddCommand(ctxGetCmd)
	ctxCmd.AddCommand(listCtxCmd)
	rootCmd.AddCommand(ctxCmd)
	// REFACTOR: call command from their respective package
	rootCmd.AddCommand(version.Command())
	rootCmd.AddCommand(kflexInit.Command())
	rootCmd.AddCommand(adopt.Command())
	rootCmd.AddCommand(delete.Command())
	rootCmd.AddCommand(create.Command())
	rootCmd.AddCommand(list.Command())
}

// TODO - work on passing the verbosity to the logger

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
