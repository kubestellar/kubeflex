package version

import (
	"fmt"
	"os"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "version",
		Short: "Provide version info",
		Long:  `Provide kubeflex version info for CLI`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			ExecuteVersion(kubeconfig, common.Version, common.BuildDate)
		},
	}
	return command
}

func ExecuteVersion(kubeconfig string, version string, buildDate string) {
	fmt.Printf("Kubeflex version: %s %s\n", version, buildDate)
	kubeVersionInfo, err := util.GetKubernetesClusterVersionInfo(kubeconfig)
	if err != nil {
		fmt.Printf("Could not connect to a Kubernetes cluster: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Kubernetes version: %s\n", kubeVersionInfo)
}
