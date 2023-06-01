package init

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mcc.ibm.org/kubeflex/pkg/client"
	"mcc.ibm.org/kubeflex/pkg/helm"
)

func Init(ctx context.Context, kubeconfig string) {
	ensureSystemNamespace(kubeconfig, ChartNamespace)
	ensureSystemDB(ctx)
}

func ensureSystemDB(ctx context.Context) {
	h := &helm.HelmHandler{
		URL:         URL,
		RepoName:    RepoName,
		ChartName:   ChartName,
		Namespace:   ChartNamespace,
		ReleaseName: ReleaseName,
		Args:        Args,
	}
	err := helm.Init(ctx, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing helm: %v\n", err)
		os.Exit(1)
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing chart: %v\n", err)
			os.Exit(1)
		}
	}
}

func ensureSystemNamespace(kubeconfig, namespace string) {
	client := client.GetClientSet(kubeconfig)

	_, err := client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err = client.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating system namespace: %v\n", err)
				os.Exit(1)
			}
		}
	}

}
