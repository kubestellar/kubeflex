package delete

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tenancyv1alpha1 "mcc.ibm.org/kubeflex/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"mcc.ibm.org/kubeflex/cmd/kflex/common"
	kfclient "mcc.ibm.org/kubeflex/pkg/client"
	"mcc.ibm.org/kubeflex/pkg/kubeconfig"
)

type CPDelete struct {
	common.CP
}

func (c *CPDelete) Delete() {
	cp := c.generateControlPlane()

	kconf, err := kubeconfig.LoadKubeconfig(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %s\n", err)
		os.Exit(1)
	}

	if err = kubeconfig.DeleteContext(kconf, c.Name); err != nil {
		fmt.Fprintf(os.Stderr, "no kubeconfig context for %s was found: %s\n", c.Name, err)
	}

	if err = kubeconfig.SwitchToInitialContext(kconf, true); err != nil {
		fmt.Fprintf(os.Stderr, "no initial kubeconfig context was found: %s\n", err)
	}

	if err = kubeconfig.WriteKubeconfig(c.Ctx, kconf); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing kubeconfig: %s\n", err)
		os.Exit(1)
	}

	cl := kfclient.GetClient(c.Kubeconfig)
	if err := cl.Delete(context.TODO(), cp, &client.DeleteOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting instance: %s\n", err)
		os.Exit(1)
	}
}

func (c *CPDelete) generateControlPlane() *tenancyv1alpha1.ControlPlane {
	return &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.Name,
		},
	}
}
