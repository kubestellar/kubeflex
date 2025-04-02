/*
Copyright 2023 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/openshift/client-go/security/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

func GetClientSet(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := getConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating clientset: %s", err)
	}
	return clientset, nil
}

func GetClient(kubeconfig string) (client.Client, error) {
	config, err := getConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTPClient: %s", err)
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(config, httpClient)
	if err != nil {
		return nil, fmt.Errorf("error creating NewDiscoveryRESTMapper: %s", err)
	}
	if err := tenancyv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding to schema: %s", err)
	}
	c, err := client.New(config, client.Options{Scheme: scheme, Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %s", err)
	}
	return c, nil
}

func GetOpendShiftSecClient(kubeconfig string) (*versioned.Clientset, error) {
	config, err := getConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return versioned.NewForConfig(config)
}

func getConfig(kubeconfig string) (*rest.Config, error) {
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
		return nil, fmt.Errorf("error building kubeconfig: %s", err)
	}
	return config, nil
}
