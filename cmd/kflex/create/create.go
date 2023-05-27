package create

import (
	"context"
	"fmt"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tenancyv1alpha1 "mcc.ibm.org/kubeflex/api/v1alpha1"

	//"sigs.k8s.io/controller-runtime/pkg/client"

	"mcc.ibm.org/kubeflex/pkg/certs"
	kfclient "mcc.ibm.org/kubeflex/pkg/client"
	"mcc.ibm.org/kubeflex/pkg/kubeconfig"
)

type CP struct {
	Ctx        context.Context
	Kubeconfig string
	Name       string
}

func (c *CP) Create() {

	//cl := kfclient.GetClient(c.Kubeconfig)

	//cp := c.generateControlPlane()

	// if err := cl.Create(context.TODO(), cp, &client.CreateOptions{}); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error creating instance: %v\n", err)
	// 	os.Exit(1)
	// }

	clientset := kfclient.GetClientSet(c.Kubeconfig)
	kubeconfig.WatchForSecretCreation(clientset, c.Name, certs.AdminConfSecret)

	if err := kubeconfig.LoadAndMerge(c.Ctx, clientset, c.Name); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading and merging kubeconfig: %v\n", err)
		os.Exit(1)
	}
}

func (c *CP) generateControlPlane() *tenancyv1alpha1.ControlPlane {
	return &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: c.Name,
		},
	}
}
