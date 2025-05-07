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
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/cmd/kflex/adopt"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cr "github.com/kubestellar/kubeflex/cmd/kflex/create"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	del "github.com/kubestellar/kubeflex/cmd/kflex/delete"
	in "github.com/kubestellar/kubeflex/cmd/kflex/init"
	cluster "github.com/kubestellar/kubeflex/cmd/kflex/init/cluster"
	"github.com/kubestellar/kubeflex/cmd/kflex/list"
	"github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var createkind bool
var kubeconfig string
var verbosity int
var Version string
var BuildDate string
var CType string
var BkType string
var Hook string
var domain string
var externalPort int
var chattyStatus bool
var hookVars []string
var hostContainer string
var overwriteExistingCtx bool
var setCurrentCtxAsHosting bool
var adoptedKubeconfig string
var adoptedContext string
var adoptedURLOverride string
var adoptedTokenExpirationSeconds int64

// defaults
const BKTypeDefault = string(tenancyv1alpha1.BackendDBTypeShared)
const CTypeDefault = string(tenancyv1alpha1.ControlPlaneTypeK8S)

var rootCmd = &cobra.Command{
	Use:   "kflex",
	Short: "CLI for kubeflex",
	Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
}

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

		ctx := createContext()
		if createkind {
			if isOCP {
				fmt.Fprintf(os.Stderr, "OpenShift cluster detected on existing context\n")
				fmt.Fprintf(os.Stdout, "Switch to a non-OpenShift context with `kubectl config use-context <context-name>` and retry.\n")
				os.Exit(1)
			}
			cluster.CreateKindCluster(chattyStatus)
		}
		in.Init(ctx, kubeconfig, Version, BuildDate, domain, strconv.Itoa(externalPort), hostContainer, chattyStatus, isOCP)
		wg.Wait()
	},
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a control plane instance",
	Long: `Create a control plane instance and switches the Kubeconfig context to
	        the current instance`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cp := cr.CPCreate{
			CP: common.CP{
				Ctx:        createContext(),
				Name:       args[0],
				Kubeconfig: kubeconfig,
			},
		}
		if CType == "" {
			CType = CTypeDefault
		}
		if BkType == "" {
			BkType = BKTypeDefault
		}
		// create passing the control plane type and backend type
		cp.Create(CType, BkType, Hook, hookVars, chattyStatus)
	},
}

var adoptCmd = &cobra.Command{
	Use:   "adopt <name>",
	Short: "Adopt a control plane from an external cluster",
	Long: `Adopt a control plane from an external cluster and switches the Kubeconfig context to
	        the current instance`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cp := adopt.CPAdopt{
			CP: common.CP{
				Ctx:        createContext(),
				Name:       args[0],
				Kubeconfig: kubeconfig,
			},
			AdoptedKubeconfig:             adoptedKubeconfig,
			AdoptedContext:                adoptedContext,
			AdoptedURLOverride:            adoptedURLOverride,
			AdoptedTokenExpirationSeconds: adoptedTokenExpirationSeconds,
		}
		// create passing the control plane type and backend type
		cp.Adopt(Hook, hookVars, chattyStatus)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a control plane instance",
	Long: `Delete a control plane instance and switches the context back to
	        the hosting cluster context`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cp := del.CPDelete{
			CP: common.CP{
				Ctx:        createContext(),
				Name:       args[0],
				Kubeconfig: kubeconfig,
			},
		}
		cp.Delete(chattyStatus)
	},
}

var ctxGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current kubeconfig context",
	Long:  `Retrieve and display the current context from the kubeconfig file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cp := cont.CPCtx{
			CP: common.CP{
				Ctx:        createContext(),
				Kubeconfig: kubeconfig,
			},
		}
		cp.GetCurrentContext()
	},
}

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
			CP: common.CP{
				Ctx:        createContext(),
				Name:       cpName,
				AliasName:  aliasName,
				Kubeconfig: kubeconfig,
			},
		}
		cp.Context(chattyStatus, true, overwriteExistingCtx, setCurrentCtxAsHosting)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all control planes",
	Long:  `List all control planes managed by KubeFlex`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cp := list.CPList{
			Ctx:        createContext(),
			Kubeconfig: kubeconfig,
		}
		cp.List(chattyStatus)
	},
}

var listCtxCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available contexts",
	Long:  `List all available contexts in the kubeconfig file`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cp := cont.CPCtx{
			CP: common.CP{
				Ctx:        createContext(),
				Kubeconfig: kubeconfig,
			},
		}
		cp.ListContexts()
	},
}

func init() {
	versionCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, and $KUBECONFIG is set, it uses the value in $KUBECONFIG, otherwise it falls back to ${HOME}/.kube/configg")
	versionCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")

	initCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, and $KUBECONFIG is set, it uses the value in $KUBECONFIG, otherwise it falls back to ${HOME}/.kube/config")
	initCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	initCmd.Flags().BoolVarP(&createkind, "create-kind", "c", false, "Create and configure a kind cluster for installing Kubeflex")
	initCmd.Flags().StringVarP(&domain, "domain", "d", "localtest.me", "domain for FQDN")
	initCmd.Flags().StringVarP(&hostContainer, "hostContainerName", "n", "kubeflex-control-plane", "Name of the hosting cluster container (kind or k3d only)")
	initCmd.Flags().IntVarP(&externalPort, "externalPort", "p", 9443, "external port used by ingress")
	initCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")

	createCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, it defaults to the value set in $KUBECONFIG, and if $KUBECONFIG is not set, it falls back to ${HOME}/.kube/config.")
	createCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	createCmd.Flags().StringVarP(&CType, "type", "t", "", "type of control plane: k8s|ocm|vcluster")
	createCmd.Flags().StringVarP(&BkType, "backend-type", "b", "", "backend DB sharing: shared|dedicated")
	createCmd.Flags().StringVarP(&Hook, "postcreate-hook", "p", "", "name of post create hook to run")
	createCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")
	createCmd.Flags().StringArrayVarP(&hookVars, "set", "e", []string{}, "set post create hook variables, in the form name=value ")

	adoptCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, it defaults to the value set in $KUBECONFIG, and if $KUBECONFIG is not set, it falls back to ${HOME}/.kube/config.")
	adoptCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	adoptCmd.Flags().StringVarP(&Hook, "postcreate-hook", "p", "", "name of post create hook to run")
	adoptCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")
	adoptCmd.Flags().StringArrayVarP(&hookVars, "set", "e", []string{}, "set post create hook variables, in the form name=value ")
	adoptCmd.Flags().StringVarP(&adoptedKubeconfig, "adopted-kubeconfig", "a", "", "path to the kubeconfig file for the adopted cluster. If unspecified, it uses the default Kubeconfig")
	adoptCmd.Flags().StringVarP(&adoptedContext, "adopted-context", "c", "", "path to adopted cluster context in adopted kubeconfig")
	adoptCmd.Flags().StringVarP(&adoptedURLOverride, "url-override", "u", "", "URL overrride for adopted cluster. Required when cluster address uses local host address, e.g. `https://127.0.0.1` ")
	adoptCmd.Flags().Int64VarP(&adoptedTokenExpirationSeconds, "expiration-seconds", "x", 86400*365, "adopted token expiration in seconds. Default is one year.")

	deleteCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, it defaults to the value set in $KUBECONFIG, and if $KUBECONFIG is not set, it falls back to ${HOME}/.kube/config.")
	deleteCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	deleteCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")

	ctxCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, it defaults to the value set in $KUBECONFIG, and if $KUBECONFIG is not set, it falls back to ${HOME}/.kube/config.")
	ctxCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	ctxCmd.Flags().BoolVarP(&chattyStatus, "chatty-status", "s", true, "chatty status indicator")
	ctxCmd.Flags().BoolVarP(&overwriteExistingCtx, "overwrite-existing-context", "o", false, "Overwrite of hosting cluster context with new control plane context")
	ctxCmd.Flags().BoolVarP(&setCurrentCtxAsHosting, "set-current-for-hosting", "c", false, "Set current context as hosting cluster context")
	ctxCmd.Flags().String("alias", "", "Set an alias name as the context, user and cluster value instead of cp name")

	ctxCmd.AddCommand(ctxGetCmd)
	ctxCmd.AddCommand(listCtxCmd)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(adoptCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(ctxCmd)
	rootCmd.AddCommand(listCmd)
}

// TODO - work on passing the verbosity to the logger
func createContext() context.Context {
	zapLogger, _ := zap.NewDevelopment(zap.AddCaller())
	logger := zapr.NewLoggerWithOptions(zapLogger)
	return logr.NewContext(context.Background(), logger)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
