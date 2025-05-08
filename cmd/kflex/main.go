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
	"strconv"
	"sync"

	"github.com/kubestellar/kubeflex/cmd/kflex/adopt"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/cmd/kflex/create"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	"github.com/kubestellar/kubeflex/cmd/kflex/delete"
	in "github.com/kubestellar/kubeflex/cmd/kflex/init"
	cluster "github.com/kubestellar/kubeflex/cmd/kflex/init/cluster"
	"github.com/kubestellar/kubeflex/cmd/kflex/list"
	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// REFACTOR: All global variables will disappear as each command package defines their own flags and retrieve then within cobra.Run function
var createkind bool   // REFACTOR: to delete
var kubeconfig string // REFACTOR: to delete
var Version string    // REFACTOR: to delete
var BuildDate string  // REFACTOR: to delete

var Hook string                         // REFACTOR: to delete
var domain string                       // REFACTOR: to delete
var externalPort int                    // REFACTOR: to delete
var chattyStatus bool                   // REFACTOR: to delete
var hookVars []string                   // REFACTOR: to delete
var hostContainer string                // REFACTOR: to delete
var overwriteExistingCtx bool           // REFACTOR: to delete
var setCurrentCtxAsHosting bool         // REFACTOR: to delete
var adoptedKubeconfig string            // REFACTOR: to delete
var adoptedContext string               // REFACTOR: to delete
var adoptedURLOverride string           // REFACTOR: to delete
var adoptedTokenExpirationSeconds int64 // REFACTOR: to delete

var rootCmd = &cobra.Command{
	Use:   "kflex",
	Short: "CLI for kubeflex",
	Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
}

// REFACTOR: all commands of kflex (non root) are moving into their own command package
// REFACTOR: to move to its own package (see how create command is implemented)
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Provide version info",
	Long:  `Provide kubeflex version info for CLI`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kubeflex version: %s %s\n", Version, BuildDate)
		kubeVersionInfo, err := util.GetKubernetesClusterVersionInfo(kubeconfig)
		if err != nil {
			fmt.Printf("Could not connect to a Kubernetes cluster: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Kubernetes version: %s\n", kubeVersionInfo)
	},
}

// REFACTOR: to move to its own package (see how create command is implemented)
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize kubeflex",
	Long:  `Installs the default shared storage backend and the kubeflex operator`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		done := make(chan bool)
		var wg sync.WaitGroup
		var isOCP bool

		util.PrintStatus("Checking if OpenShift cluster...", done, &wg, chattyStatus)
		clientsetp, err := client.GetClientSet(kubeconfig)
		if err == nil {
			isOCP = util.IsOpenShift(*clientsetp)
			if isOCP {
				done <- true
				util.PrintStatus("OpenShift cluster detected", done, &wg, chattyStatus)
			}
		}
		done <- true

		if createkind {
			if isOCP {
				fmt.Fprintf(os.Stderr, "OpenShift cluster detected on existing context\n")
				fmt.Fprintf(os.Stdout, "Switch to a non-OpenShift context with `kubectl config use-context <context-name>` and retry.\n")
				os.Exit(1)
			}
			cluster.CreateKindCluster(chattyStatus)
		}

		// REFACTOR: leverage CP struct to give Context and Kubeconfig
		cp := common.NewCP(kubeconfig)
		in.Init(cp.Ctx, cp.Kubeconfig, Version, BuildDate, domain, strconv.Itoa(externalPort), hostContainer, chattyStatus, isOCP)
		wg.Wait()
	},
}

// REFACTOR: to move to its own package (see how create command is implemented)
// REFACTOR: remove cont.CPAdopt as common.CP is enough
var adoptCmd = &cobra.Command{
	Use:   "adopt <name>",
	Short: "Adopt a control plane from an external cluster",
	Long: `Adopt a control plane from an external cluster and switches the Kubeconfig context to
	        the current instance`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cp := adopt.CPAdopt{
			CP:                            common.NewCP(kubeconfig, common.WithName(args[0])),
			AdoptedKubeconfig:             adoptedKubeconfig,
			AdoptedContext:                adoptedContext,
			AdoptedURLOverride:            adoptedURLOverride,
			AdoptedTokenExpirationSeconds: adoptedTokenExpirationSeconds,
		}
		// create passing the control plane type and backend type
		cp.Adopt(Hook, hookVars, chattyStatus)
	},
}

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
	initCmd.Flags().BoolVarP(&createkind, "create-kind", "c", false, "Create and configure a kind cluster for installing Kubeflex")
	initCmd.Flags().StringVarP(&domain, "domain", "d", "localtest.me", "domain for FQDN")
	initCmd.Flags().StringVarP(&hostContainer, "hostContainerName", "n", "kubeflex-control-plane", "Name of the hosting cluster container (kind or k3d only)")
	initCmd.Flags().IntVarP(&externalPort, "externalPort", "p", 9443, "external port used by ingress")
	// REFACTOR: to move to its own package (see how create command is implemented)
	adoptCmd.Flags().StringVarP(&Hook, "postcreate-hook", "p", "", "name of post create hook to run")
	adoptCmd.Flags().StringArrayVarP(&hookVars, "set", "e", []string{}, "set post create hook variables, in the form name=value ")
	adoptCmd.Flags().StringVarP(&adoptedKubeconfig, "adopted-kubeconfig", "a", "", "path to the kubeconfig file for the adopted cluster. If unspecified, it uses the default Kubeconfig")
	adoptCmd.Flags().StringVarP(&adoptedContext, "adopted-context", "c", "", "path to adopted cluster context in adopted kubeconfig")
	adoptCmd.Flags().StringVarP(&adoptedURLOverride, "url-override", "u", "", "URL overrride for adopted cluster. Required when cluster address uses local host address, e.g. `https://127.0.0.1` ")
	adoptCmd.Flags().Int64VarP(&adoptedTokenExpirationSeconds, "expiration-seconds", "x", 86400*365, "adopted token expiration in seconds. Default is one year.")
	// REFACTOR: to move to its own package (see how create command is implemented)
	ctxCmd.Flags().BoolVarP(&overwriteExistingCtx, "overwrite-existing-context", "o", false, "Overwrite of hosting cluster context with new control plane context")
	ctxCmd.Flags().BoolVarP(&setCurrentCtxAsHosting, "set-current-for-hosting", "c", false, "Set current context as hosting cluster context")
	ctxCmd.Flags().String("alias", "", "Set an alias name as the context, user and cluster value instead of cp name")
	// REFACTOR: to move to its own package (see how create command is implemented)
	ctxCmd.AddCommand(ctxGetCmd)
	ctxCmd.AddCommand(listCtxCmd)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(adoptCmd)
	rootCmd.AddCommand(ctxCmd)
	// REFACTOR: call command from their respective package
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
