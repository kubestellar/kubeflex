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

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cr "github.com/kubestellar/kubeflex/cmd/kflex/create"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	del "github.com/kubestellar/kubeflex/cmd/kflex/delete"
	in "github.com/kubestellar/kubeflex/cmd/kflex/init"
	cluster "github.com/kubestellar/kubeflex/cmd/kflex/init/cluster"
	initmanager "github.com/kubestellar/kubeflex/cmd/kflex/init/manager"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var createkind bool
var kubeconfig string
var verbosity int

var rootCmd = &cobra.Command{
	Use:   "kflex",
	Short: "CLI for kubeflex",
	Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize kubeflex",
	Long:  `Installs the default shared storage backend and the kubeflex operator`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := createContext()
		if createkind {
			cluster.CreateKindCluster()
		}
		in.Init(ctx, kubeconfig)
		initmanager.InstallManager()
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
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
		cp.Create()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
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
		cp.Delete()
	},
}

var ctxCmd = &cobra.Command{
	Use:   "ctx",
	Short: "switch Kubeconfig context to a control plane instance",
	Long: `Running without an argument switches the context back to the initial context,
			        while providing the control plane name as argument switches the context to
					that control plane`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cpName := ""
		if len(args) == 1 {
			cpName = args[0]
		}
		cp := cont.CPCtx{
			CP: common.CP{
				Ctx:        createContext(),
				Name:       cpName,
				Kubeconfig: kubeconfig,
			},
		}
		cp.Context()
	},
}

func init() {
	initCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	initCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity
	initCmd.Flags().BoolVarP(&createkind, "create-kind", "c", false, "Create and configure a kind cluster for installing Kubeflex")

	createCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	createCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	deleteCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	deleteCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	ctxCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "path to kubeconfig file")
	ctxCmd.Flags().IntVarP(&verbosity, "verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(ctxCmd)
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
