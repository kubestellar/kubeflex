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
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func LoadAndMerge(ctx context.Context, client kubernetes.Clientset, name, controlPlaneType string) error {
	konfig, err := LoadKubeconfig(ctx)
	if err != nil {
		return err
	}

	if controlPlaneType != string(tenancyv1alpha1.ControlPlaneTypeHost) {
		cpKonfig, err := loadControlPlaneKubeconfig(ctx, client, name, controlPlaneType)
		if err != nil {
			return err
		}
		adjustConfigKeys(cpKonfig, name, controlPlaneType)

		err = merge(konfig, cpKonfig)
		if err != nil {
			return err
		}
	} else {
		err = SwitchToHostingClusterContext(konfig, false)
		if err != nil {
			return err
		}
	}

	return WriteKubeconfig(ctx, konfig)
}

// LoadAndMergeNoWrite: works as LoadAndMerge but on supplied konfig from file and does not write it back
func LoadAndMergeNoWrite(ctx context.Context, client kubernetes.Clientset, name, controlPlaneType string, konfig *clientcmdapi.Config) error {
	cpKonfig, err := loadControlPlaneKubeconfig(ctx, client, name, controlPlaneType)
	if err != nil {
		return err
	}
	adjustConfigKeys(cpKonfig, name, controlPlaneType)

	err = merge(konfig, cpKonfig)
	if err != nil {
		return err
	}

	return nil
}

func loadControlPlaneKubeconfig(ctx context.Context, client kubernetes.Clientset, name, controlPlaneType string) (*clientcmdapi.Config, error) {
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	var ks *v1.Secret
	var errGet error
	err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 15*time.Minute, false, func(ctx context.Context) (bool, error) {
		ks, errGet = client.CoreV1().Secrets(namespace).Get(ctx,
			util.GetKubeconfSecretNameByControlPlaneType(controlPlaneType),
			metav1.GetOptions{})
		if errGet != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error waiting for control plane kubeconfig secret: %s, %s", err, errGet)
	}

	key := util.GetKubeconfSecretKeyNameByControlPlaneType(controlPlaneType)
	return clientcmd.Load(ks.Data[key])
}

func LoadKubeconfig(ctx context.Context) (*clientcmdapi.Config, error) {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.LoadFromFile(kubeconfig)
}

func WriteKubeconfig(ctx context.Context, config *clientcmdapi.Config) error {
	kubeconfig := clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	return clientcmd.WriteToFile(*config, kubeconfig)
}

func WatchForSecretCreation(clientset kubernetes.Clientset, controlPlaneName, secretName string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(controlPlaneName)

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

func WaitForNamespaceReady(ctx context.Context, clientset kubernetes.Interface, controlPlaneName string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(controlPlaneName)

	err := wait.PollUntilContextTimeout(
		ctx,
		2*time.Second,
		2*time.Minute,
		true,
		func(context.Context) (bool, error) {
			ns, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return false, nil // Retry if namespace is not found
			} else if err != nil {
				return false, fmt.Errorf("error checking namespace status: %v", err)
			}

			if ns.Status.Phase == v1.NamespaceActive {
				return true, nil // Namespace is ready
			}

			return false, nil // Continue waiting
		},
	)

	if err != nil {
		return fmt.Errorf("timed out waiting for namespace %s to be ready: %v", namespace, err)
	}
	return nil
}

func adjustConfigKeys(config *clientcmdapi.Config, cpName, controlPlaneType string) {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeOCM):
		renameKey(config.Clusters, "multicluster-controlplane", certs.GenerateClusterName(cpName))
		renameKey(config.AuthInfos, "user", certs.GenerateAuthInfoAdminName(cpName))
		renameKey(config.Contexts, "multicluster-controlplane", certs.GenerateContextName(cpName))
		config.CurrentContext = certs.GenerateContextName(cpName)
		config.Contexts[certs.GenerateContextName(cpName)] = &clientcmdapi.Context{
			Cluster:  certs.GenerateClusterName(cpName),
			AuthInfo: certs.GenerateAuthInfoAdminName(cpName),
		}
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		renameKey(config.Clusters, "my-vcluster", certs.GenerateClusterName(cpName))
		renameKey(config.AuthInfos, "my-vcluster", certs.GenerateAuthInfoAdminName(cpName))
		renameKey(config.Contexts, "my-vcluster", certs.GenerateContextName(cpName))
		config.CurrentContext = certs.GenerateContextName(cpName)
		config.Contexts[certs.GenerateContextName(cpName)] = &clientcmdapi.Context{
			Cluster:  certs.GenerateClusterName(cpName),
			AuthInfo: certs.GenerateAuthInfoAdminName(cpName),
		}
	default:
		return
	}
}

func renameKey(m interface{}, oldKey string, newKey string) interface{} {
	switch v := m.(type) {
	case map[string]*clientcmdapi.Cluster:
		if cluster, ok := v[oldKey]; ok {
			delete(v, oldKey)
			v[newKey] = cluster
		}
	case map[string]*clientcmdapi.AuthInfo:
		if authInfo, ok := v[oldKey]; ok {
			delete(v, oldKey)
			v[newKey] = authInfo
		}
	case map[string]*clientcmdapi.Context:
		if context, ok := v[oldKey]; ok {
			delete(v, oldKey)
			v[newKey] = context
		}
	default:
		// no action
	}
	return m
}
