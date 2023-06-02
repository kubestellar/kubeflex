package init

import (
	"context"
	"fmt"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mcc.ibm.org/kubeflex/pkg/client"
	"mcc.ibm.org/kubeflex/pkg/helm"
	"mcc.ibm.org/kubeflex/pkg/util"
)

func Init(ctx context.Context, kubeconfig string) {
	done := make(chan bool)
	var wg sync.WaitGroup

	util.PrintStatus("Installing shared backend DB...", done, &wg)

	ensureSystemNamespace(kubeconfig, util.DBNamespace)

	ensureSystemDB(ctx)
	done <- true

	util.PrintStatus("Waiting for shared backend DB to become ready...", done, &wg)
	util.WaitForStatefulSetReady(
		client.GetClientSet(kubeconfig),
		util.GeneratePSReplicaSetName(util.DBReleaseName),
		util.DBNamespace)

	done <- true
	wg.Wait()
}

func ensureSystemDB(ctx context.Context) {
	h := &helm.HelmHandler{
		URL:         URL,
		RepoName:    RepoName,
		ChartName:   ChartName,
		Namespace:   util.DBNamespace,
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
