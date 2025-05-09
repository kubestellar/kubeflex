package ctx

import (
	"fmt"
	"os"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"github.com/spf13/cobra"
)

func CommandGet() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Get the current kubeconfig context",
		Long:  `Retrieve and display the current context from the kubeconfig file`,
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			kubeconfig, _ := cmd.Flags().GetString(common.KubeconfigFlag)
			cp := common.NewCP(kubeconfig)
			ExecuteCtxGet(cp)
		},
	}
	return command
}

func ExecuteCtxGet(cp common.CP) {
	currentContext, err := kubeconfig.GetCurrentContext(cp.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error retrieving current context: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(currentContext)
}
