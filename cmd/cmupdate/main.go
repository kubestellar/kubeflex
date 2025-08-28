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
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubestellar/kubeflex/pkg/reconcilers/ocm"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	configMapName            = "cluster-info"
	keyToUpdate              = "kubeconfig"
	kubePublicNamespace      = "kube-public"
	clusterName              = ""
	defaultHostContainerName = "kubeflex-control-plane"
	BootstrapConfigMap       = "cluster-info"
	GeneratedCLusterInfoKey  = "kubestellar.io/generated"
)

// The CM update updates the cluster-info config map in a OCM control plane
// for setting the internal server value that can be used by other kind servers
// on the same docker network
func main() {
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		log.Fatal("Namespace not found.")
	} else {
		log.Printf("Current Namespace: %s", namespace)
	}
	hostContainer := os.Getenv("HOST_CONTAINER")
	if hostContainer == "" {
		hostContainer = defaultHostContainerName
	}
	log.Printf("Using hostContainer: %s", hostContainer)
	baseURL := fmt.Sprintf("https://%s", hostContainer)

	externalURL := os.Getenv("EXTERNAL_URL")
	if externalURL != "" {
		log.Printf("Using external URL: %s", externalURL)
	}
	kubeconfigSecret := os.Getenv("KUBECONFIG_SECRET")
	if kubeconfigSecret == "" {
		log.Fatal("KUBECONFIG_SECRET name not found.")
	} else {
		log.Printf("Using kubeconfigSecret: %s", kubeconfigSecret)
	}
	kubeconfigSecretKey := os.Getenv("KUBECONFIG_SECRET_KEY")
	if kubeconfigSecretKey == "" {
		log.Fatal("KUBECONFIG_SECRET_KEY not found.")
	} else {
		log.Printf("Using kubeconfigSecretKey: %s", kubeconfigSecretKey)
	}
	ctx := context.TODO()

	// Create the Kubernetes clientset
	config := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// wait for API server to be available
	log.Printf("Waiting for API server to become available...")
	var serviceName string
	var nodePortServiceName string
	switch kubeconfigSecret {
	case util.VClusterKubeConfigSecret:
		nodePortServiceName = util.VClusterNodePortServiceName
		serviceName = util.VClusterServiceName
		if err := util.WaitForStatefulSetReady(*clientset,
			util.VClusterServerDeploymentName,
			namespace); err != nil {
			log.Fatalf("Error waiting for stateful set to become ready: %s", err)
		}
	case util.OCMKubeConfigSecret:
		serviceName = ocm.ServiceName
		nodePortServiceName = ocm.ServiceName
		if err := util.WaitForDeploymentReady(*clientset,
			util.OCMServerDeploymentName,
			namespace); err != nil {
			log.Fatalf("Error waiting for deployment to become ready: %s", err)
		}
	default:
		log.Fatal("Unknown control plane type")
	}
	log.Printf("API server ready.")

	nodePort, err := retrieveServiceNodePort(clientset, nodePortServiceName, namespace, ctx)
	if err != nil {
		log.Fatalf("Error looking up the node port from ocm service: %s", err)
	}

	configData, err := retrieveHostedServerKubeConfig(clientset, namespace, ctx, kubeconfigSecret, kubeconfigSecretKey)
	if err != nil {
		log.Fatalf("Error getting kubeconfig for hosted server: %s", err)
	}

	clientForHostedServer, err := createClientForHostedServer(configData, namespace, serviceName)
	if err != nil {
		log.Fatalf("Error getting client for for hosted server: %s", err)
	}

	createOnly := false
	configMap, err := retrieveConfigMap(clientForHostedServer)
	if err != nil {
		if apierrors.IsNotFound(err) {
			configMap, err = createClusterInfo(clientForHostedServer, configData)
			if err != nil {
				log.Fatalf("Error creating cluster-info map: %s", err)
			}
			createOnly = true
		} else {
			log.Fatalf("Error retrieving the cluster-info map for for hosted server: %s", err)
		}
	}

	serverURL := fmt.Sprintf("%s:%d", baseURL, nodePort)
	if externalURL != "" {
		serverURL = fmt.Sprintf("https://%s", externalURL)
	}
	updatedConfigMap := updateConfigMap(configMap, clusterName, keyToUpdate, serverURL)

	err = createOrUpdateConfigMapValue(clientForHostedServer, ctx, updatedConfigMap, createOnly)
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
		if port.Name == shared.DefaultPortName {
			return port.NodePort, nil
		}
	}
	return -1, fmt.Errorf("port not found")
}

func retrieveHostedServerKubeConfig(clientset *kubernetes.Clientset, ns string, ctx context.Context, kubeconfigSecret, kubeconfigSecretKey string) ([]byte, error) {
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

func createClientForHostedServer(configData []byte, namespace, serviceName string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.NewClientConfigFromBytes(configData)
	if err != nil {
		return nil, err
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	if util.IsInCluster() {
		restConfig.Host = fmt.Sprintf("%s.%s", serviceName, namespace)
	}

	return kubernetes.NewForConfig(restConfig)
}

func retrieveConfigMap(clientset *kubernetes.Clientset) (*corev1.ConfigMap, error) {
	log.Printf("Retrieving config map...")
	var configMap *corev1.ConfigMap
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// need to wait as even with API server in ready state it may not connect right away
	err = wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		configMap, err = clientset.CoreV1().ConfigMaps(kubePublicNamespace).Get(ctx, configMapName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, err
			}
			return false, nil
		}
		return true, nil
	})
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

func createOrUpdateConfigMapValue(clientset *kubernetes.Clientset, ctx context.Context, updatedConfigMap *corev1.ConfigMap, createOnly bool) error {
	// if create only, the config map was found and can just create
	if createOnly {
		if _, err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Create(ctx, updatedConfigMap, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}

	// if the map was found, check if immutable - we need to update even when set to immutable, which OCM installer does.
	if updatedConfigMap.Immutable != nil && *updatedConfigMap.Immutable {
		if err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Delete(ctx, updatedConfigMap.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
		// need to remove runtime fields to do a create
		cm := util.ZeroFields(updatedConfigMap)
		if _, err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Create(ctx, cm.(*corev1.ConfigMap), metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}
	_, err := clientset.CoreV1().ConfigMaps(kubePublicNamespace).Update(ctx, updatedConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func getHostedClusterConfig(configData []byte) (*clientcmdapi.Cluster, error) {
	config, err := clientcmd.Load(configData)
	if err != nil {
		return nil, err
	}
	context := config.CurrentContext

	currentCtx, exists := config.Contexts[context]
	if !exists {
		return nil, errors.Errorf("failed to find the given Current Context in Contexts of the kubeconfig")
	}
	currentCluster, exists := config.Clusters[currentCtx.Cluster]
	if !exists {
		return nil, errors.Errorf("failed to find the given CurrentContext Cluster in Clusters of the kubeconfig")
	}
	return currentCluster, nil
}

// createClusterInfo will create a ConfigMap named cluster-info in the kube-public namespace.
func createClusterInfo(client kubernetes.Interface, configData []byte) (*corev1.ConfigMap, error) {
	cluster, err := getHostedClusterConfig(configData)
	if err != nil {
		return nil, err
	}

	kubeconfig := &clientcmdapi.Config{Clusters: map[string]*clientcmdapi.Cluster{"": cluster}}
	if err := clientcmdapi.FlattenConfig(kubeconfig); err != nil {
		return nil, err
	}
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return nil, err
	}
	clusterInfo := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BootstrapConfigMap,
			Namespace: metav1.NamespacePublic,
			Annotations: map[string]string{
				GeneratedCLusterInfoKey: "true",
			},
		},
		Immutable: pointer.Bool(true),
		Data: map[string]string{
			"kubeconfig": string(kubeconfigBytes),
		},
	}
	return clusterInfo, nil
}
