package kubeconfig

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	//"k8s.io/client-go/util/workqueue"

	"mcc.ibm.org/kubeflex/pkg/certs"
	"mcc.ibm.org/kubeflex/pkg/util"
)

func LoadAndMerge(ctx context.Context, client kubernetes.Clientset, name string) error {
	cpKonfig, err := loadControlPlaneKubeconfig(ctx, client, name)
	if err != nil {
		return err
	}

	konfig, err := loadKubeconfig(ctx)
	if err != nil {
		return err
	}

	// Merge the two configs together.
	err = merge(konfig, cpKonfig)
	if err != nil {
		return err
	}

	return writeKubeconfig(ctx, konfig)
}

func loadControlPlaneKubeconfig(ctx context.Context, client kubernetes.Clientset, name string) (*clientcmdapi.Config, error) {
	fmt.Println("in loadControlPlaneKubeconfig")

	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	fmt.Printf("before  Get secret name=%s namespace=%s ctx=%#v \n", name, namespace, ctx)
	ks, err := client.CoreV1().Secrets(namespace).Get(ctx, certs.AdminConfSecret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	fmt.Print("afetr getting the secret")

	return clientcmd.Load(ks.Data[certs.ConfSecretKey])
}

func loadKubeconfig(ctx context.Context) (*clientcmdapi.Config, error) {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.LoadFromFile(kubeconfig)
}

func writeKubeconfig(ctx context.Context, config *clientcmdapi.Config) error {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.WriteToFile(*config, kubeconfig)
}

func WatchForSecretCreation(clientset kubernetes.Clientset, name, secretName string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(name)
	// create the listwatch object
	listwatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"secrets",
		namespace,
		fields.Everything(),
	)

	// create the stop channel
	stopCh := make(chan struct{})

	// create the informer
	_, controller := cache.NewInformer(
		listwatch,
		&v1.Secret{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				secret := obj.(*v1.Secret)
				if secret.Name == secretName {
					close(stopCh)
				}
			},
		},
	)

	// start the informer
	go controller.Run(stopCh)
	<-stopCh
	return nil
}
