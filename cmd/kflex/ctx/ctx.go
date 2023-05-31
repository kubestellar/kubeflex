package ctx

import (
	"fmt"
	"os"

	"mcc.ibm.org/kubeflex/cmd/kflex/common"
	"mcc.ibm.org/kubeflex/pkg/kubeconfig"
)

type CPCtx struct {
	common.CP
}

// this is used to switch context
func (c *CPCtx) Context() {
	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	switch c.CP.Name {
	case "":
		if err = kubeconfig.SwitchToInitialContext(kconf, false); err != nil {
			fmt.Fprintf(os.Stderr, "Error switching kubeconfig to initial context: %s\n", err)
			os.Exit(1)
		}
	default:
		if err = kubeconfig.SwitchContext(kconf, c.Name); err != nil {
			fmt.Fprintf(os.Stderr, "Error switching kubeconfig context: %s\n", err)
			os.Exit(1)
		}
	}

	if err = kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}
}
