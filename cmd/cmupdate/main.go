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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubestellar/kubeflex/pkg/reconcilers/ocm"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	kubeconfigSecret    = "multicluster-controlplane-kubeconfig"
	kubeconfigSecretKey = "kubeconfig"
	configMapName       = "cluster-info"
	keyToUpdate         = "kubeconfig"
	kubePublicNamespace = "kube-public"
	controlPlaneType    = "ocm"
	clusterName         = ""
	baseURL             = "https://kubeflex-control-plane"
)

// The CM update updates the cluster-info config map in a OCM control plane
// for setting the internal server value that can be used by other kind servers
// on the same docker network
func main() {
	kconfig := flag.String("kconfig", "", "path to the kubeconfig file")
	flag.Parse()
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		log.Fatal("Namespace not found.")
	} else {
		log.Printf("Current Namespace: %s", namespace)
	}
	ctx := context.TODO()

	// Create the Kubernetes clientset
	config, err := clientcmd.BuildConfigFromFlags("", *kconfig)
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// wait for ocm deployment to be available
	if err := util.WaitForDeploymentReady(*clientset,
		util.OCMServerDeploymentName,
		namespace); err != nil {

		log.Fatalf("Error waiting for deployment to become ready: %s", err)
	}

	nodePort, err := retrieveServiceNodePort(clientset, ocm.ServiceName, namespace, ctx)
	if err != nil {
		log.Fatalf("Error looking up the node port from ocm service: %s", err)
	}

	configData, err := retrieveHostedServerKubeConfig(clientset, namespace, ctx)
	if err != nil {
		log.Fatalf("Error getting kubeconfig for hosted server: %s", err)
	}

	clientForHostedServer, err := createClientForHostedServer(configData, namespace)
	if err != nil {
		log.Fatalf("Error getting client for for hosted server: %s", err)
	}

	configMap, err := retrieveConfigMap(clientForHostedServer, ctx)
	if err != nil {
		log.Fatalf("Error retrieving the cluster-info map for for hosted server: %s", err)
	}

	serverURL := fmt.Sprintf("%s:%d", baseURL, nodePort)
	updatedConfigMap := updateConfigMap(configMap, clusterName, keyToUpdate, serverURL)

	err = updateConfigMapValue(clientForHostedServer, ctx, updatedConfigMap)
	if err != nil {
		log.Fatalf("Error updating the cluster-info map for for hosted server: %s", err)
	}

	log.Println("ConfigMap updated successfully!")
}

func retrieveServiceNodePort(clientset *kubernetes.Clientset, name, ns string, ctx context.Context) (int32, error) {
	svc, err := clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return -1, err
	}
	for _, port := range svc.Spec.Ports {
		if port.Name == "https" {
			return port.NodePort, nil
		}
	}
	return -1, fmt.Errorf("port not found")
}

func retrieveHostedServerKubeConfig(clientset *kubernetes.Clientset, ns string, ctx context.Context) ([]byte, error) {
	secret, err := clientset.CoreV1().Secrets(ns).Get(ctx, kubeconfigSecret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	kData, ok := secret.Data[kubeconfigSecretKey]
	if !ok {
		return nil, fmt.Errorf("key %s not found in kubeconfig secret", kubeconfigSecretKey)
	}
	return kData, nil
}

func createClientForHostedServer(configData []byte, namespace string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.NewClientConfigFromBytes(configData)
	if err != nil {
		return nil, err
	}

	restConfig, err := config.ClientConfig()
	restConfig.Host = fmt.Sprintf("multicluster-controlplane.%s:9444", namespace)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restConfig)
}

func retrieveConfigMap(clientset *kubernetes.Clientset, ctx context.Context) (*corev1.ConfigMap, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

func updateConfigMap(configMap *corev1.ConfigMap, clusterName, keyToUpdate, newServerValue string) *corev1.ConfigMap {
	configData := configMap.Data[keyToUpdate]
	config, err := clientcmd.Load([]byte(configData))
	if err != nil {
		panic(err.Error())
	}

	// Update the server value in the Config object
	config.Clusters[clusterName].Server = newServerValue

	updatedConfigData, err := clientcmd.Write(*config)
	if err != nil {
		panic(err.Error())
	}

	// Create a copy of the original config map and update its data field
	updatedConfigMap := configMap.DeepCopy()
	updatedConfigMap.Data[keyToUpdate] = string(updatedConfigData)

	return updatedConfigMap
}

func updateConfigMapValue(clientset *kubernetes.Clientset, ctx context.Context, updatedConfigMap *corev1.ConfigMap) error {
	_, err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Update(ctx, updatedConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
