package client

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	tenancyv1alpha1 "mcc.ibm.org/kubeflex/api/v1alpha1"
)

func GetClientSet(kubeconfig string) kubernetes.Clientset {
	config := getConfig(kubeconfig)
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating clientset: %v\n", err)
		os.Exit(1)
	}
	return *clientset
}

func GetClient(kubeconfig string) client.Client {
	config := getConfig(kubeconfig)
	// add the scheme for your custom API
	scheme := runtime.NewScheme()
	mapper, err := apiutil.NewDiscoveryRESTMapper(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating NewDiscoveryRESTMapper: %v\n", err)
		os.Exit(1)
	}
	if err := tenancyv1alpha1.AddToScheme(scheme); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding to schema: %v\n", err)
		os.Exit(1)
	}
	c, err := client.New(config, client.Options{Scheme: scheme, Mapper: mapper})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}
	return c
}

func getConfig(kubeconfig string) *rest.Config {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := homedir.Dir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
				os.Exit(1)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}
	return config
}
