package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"mcc.ibm.org/kubeflex/cmd/kflex/common"
	cr "mcc.ibm.org/kubeflex/cmd/kflex/create"
	cont "mcc.ibm.org/kubeflex/cmd/kflex/ctx"
	del "mcc.ibm.org/kubeflex/cmd/kflex/delete"
	in "mcc.ibm.org/kubeflex/cmd/kflex/init"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "kflex",
		Short: "CLI for kubeflex",
		Long:  `A flexible and scalable solution for running Kubernetes control plane APIs`,
	}

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize kubeflex",
		Long:  `Installs the default storage backend and the kubeflex operator`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := createContext()
			kubeconfig := ""
			if cmd.Flags().Lookup("kubeconfig").Changed {
				kubeconfig = cmd.Flag("kubeconfig").Value.String()
			}
			in.Init(ctx, kubeconfig)
		},
	}

	initCmd.Flags().StringP("kubeconfig", "k", "", "path to kubeconfig file")
	initCmd.Flags().IntP("verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a control plane instance",
		Long:  `Create a control plane instance`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cp := cr.CPCreate{
				CP: common.CP{
					Ctx:  createContext(),
					Name: args[0],
				},
			}
			if cmd.Flags().Lookup("kubeconfig").Changed {
				cp.Kubeconfig = cmd.Flag("kubeconfig").Value.String()
			}
			cp.Create()
		},
	}

	createCmd.Flags().StringP("kubeconfig", "k", "", "path to kubeconfig file")
	createCmd.Flags().IntP("verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a control plane instance",
		Long:  `Delete a control plane instance`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cp := del.CPDelete{
				CP: common.CP{
					Ctx:  createContext(),
					Name: args[0],
				},
			}
			if cmd.Flags().Lookup("kubeconfig").Changed {
				cp.Kubeconfig = cmd.Flag("kubeconfig").Value.String()
			}
			cp.Delete()
		},
	}

	deleteCmd.Flags().StringP("kubeconfig", "k", "", "path to kubeconfig file")
	deleteCmd.Flags().IntP("verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

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
					Ctx:  createContext(),
					Name: cpName,
				},
			}
			if cmd.Flags().Lookup("kubeconfig").Changed {
				cp.Kubeconfig = cmd.Flag("kubeconfig").Value.String()
			}
			cp.Context()
		},
	}

	ctxCmd.Flags().StringP("kubeconfig", "k", "", "path to kubeconfig file")
	ctxCmd.Flags().IntP("verbosity", "v", 0, "log level") // TODO - figure out how to inject verbosity

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(ctxCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createContext() context.Context {
	flag.Parse()
	zapLogger, _ := zap.NewDevelopment(zap.AddCaller())
	logger := zapr.NewLoggerWithOptions(zapLogger)
	return logr.NewContext(context.Background(), logger)
}
