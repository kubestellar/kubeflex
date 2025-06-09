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
	"os"

	"github.com/fatih/color"
	"github.com/kubestellar/kubeflex/cmd/kflex/adopt"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/cmd/kflex/create"
	kflexCtx "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	"github.com/kubestellar/kubeflex/cmd/kflex/delete"
	kflexInit "github.com/kubestellar/kubeflex/cmd/kflex/init"
	"github.com/kubestellar/kubeflex/cmd/kflex/list"
	"github.com/kubestellar/kubeflex/cmd/kflex/version"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var rootCmd = &cobra.Command{
	Use:   "kflex",
	Short: "CLI for kubeflex",
	Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
}

func init() {
	pflagset := rootCmd.PersistentFlags()
	pflagset.StringP(common.KubeconfigFlag, "k", clientcmd.RecommendedHomeFile, "path to the kubeconfig file for the KubeFlex hosting cluster. If not specified, and $KUBECONFIG is set, it uses the value in $KUBECONFIG, otherwise it falls back to ${HOME}/.kube/config")
	pflagset.BoolP(common.ChattyStatusFlag, "s", true, "chatty status indicator")
	pflagset.IntP(common.VerbosityFlag, "v", 0, "log level") // TODO - figure out how to inject verbosity
	rootCmd.AddCommand(kflexCtx.Command())
	rootCmd.AddCommand(version.Command())
	rootCmd.AddCommand(kflexInit.Command())
	rootCmd.AddCommand(adopt.Command())
	rootCmd.AddCommand(delete.Command())
	rootCmd.AddCommand(create.Command())
	rootCmd.AddCommand(list.Command())
}

// TODO - work on passing the verbosity to the logger
func main() {
	// TODO - find a way to inject it using Makefile
	common.WarningMessage = "WARNING: current kflex version introduces BREAKING CHANGES related to kflex and your kubeconfig file which may interrupt kflex to function properly.\nSee https://github.com/kubestellar/kubeflex/blob/main/docs/users.md"
	if common.WarningMessage != "" {
		color.Yellow(common.WarningMessage)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
