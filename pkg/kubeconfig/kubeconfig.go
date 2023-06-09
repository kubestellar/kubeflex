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

package kubeconfig

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func LoadAndMerge(ctx context.Context, client kubernetes.Clientset, name string) error {
	cpKonfig, err := loadControlPlaneKubeconfig(ctx, client, name)
	if err != nil {
		return err
	}

	konfig, err := LoadKubeconfig(ctx)
	if err != nil {
		return err
	}

	err = merge(konfig, cpKonfig)
	if err != nil {
		return err
	}

	return WriteKubeconfig(ctx, konfig)
}

func loadControlPlaneKubeconfig(ctx context.Context, client kubernetes.Clientset, name string) (*clientcmdapi.Config, error) {
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	ks, err := client.CoreV1().Secrets(namespace).Get(ctx, certs.AdminConfSecret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(ks.Data[certs.ConfSecretKey])
}

func LoadKubeconfig(ctx context.Context) (*clientcmdapi.Config, error) {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.LoadFromFile(kubeconfig)
}

func WriteKubeconfig(ctx context.Context, config *clientcmdapi.Config) error {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.WriteToFile(*config, kubeconfig)
}

func WatchForSecretCreation(clientset kubernetes.Clientset, name, secretName string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	listwatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"secrets",
		namespace,
		fields.Everything(),
	)

	stopCh := make(chan struct{})

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

	go controller.Run(stopCh)
	<-stopCh
	return nil
}
