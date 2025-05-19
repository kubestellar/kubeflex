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
	"encoding/json"
	"fmt"
	"time"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	ExtensionConfigName             = "kflex-config-extension-name"
	ExtensionInitialContextName     = "kflex-initial-ctx-name"
	ExtensionControlPlaneName       = "kflex-controlplane-name"
	ExtensionKubeflexKey            = "kubeflex"
	ExtensionLabelManageByKubeflex  = "kubeflex.dev/is-managed"
	ControlPlaneTypeOCMDefault      = "multicluster-controlplane"
	ControlPlaneTypeVClusterDefault = "my-vcluster"
)

func unMarshallCM(obj runtime.Object) (*corev1.ConfigMap, error) {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshaling object %s", err)
	}
	cm := corev1.ConfigMap{}
	json.Unmarshal(jsonData, &cm)
	return &cm, nil
}

func adjustConfigKeys(config *clientcmdapi.Config, cpName, controlPlaneType string) {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeOCM):
		RenameKey(config.Clusters, ControlPlaneTypeOCMDefault, certs.GenerateClusterName(cpName))
		RenameKey(config.AuthInfos, "user", certs.GenerateAuthInfoAdminName(cpName))
		RenameKey(config.Contexts, ControlPlaneTypeOCMDefault, certs.GenerateContextName(cpName))
		config.CurrentContext = certs.GenerateContextName(cpName)
		config.Contexts[certs.GenerateContextName(cpName)] = &clientcmdapi.Context{
			Cluster:  certs.GenerateClusterName(cpName),
			AuthInfo: certs.GenerateAuthInfoAdminName(cpName),
		}
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		RenameKey(config.Clusters, ControlPlaneTypeVClusterDefault, certs.GenerateClusterName(cpName))
		RenameKey(config.AuthInfos, ControlPlaneTypeVClusterDefault, certs.GenerateAuthInfoAdminName(cpName))
		RenameKey(config.Contexts, ControlPlaneTypeVClusterDefault, certs.GenerateContextName(cpName))
		config.CurrentContext = certs.GenerateContextName(cpName)
		config.Contexts[certs.GenerateContextName(cpName)] = &clientcmdapi.Context{
			Cluster:  certs.GenerateClusterName(cpName),
			AuthInfo: certs.GenerateAuthInfoAdminName(cpName),
		}
	default:
		return
	}
}

// Load kubeconfig from the control plane (server-side)
func loadKubeconfigFromControlPlane(ctx context.Context, client kubernetes.Clientset, name, controlPlaneType string) (*clientcmdapi.Config, error) {
	var kubeconfigSecret *corev1.Secret
	var errGet error
	namespace := util.GenerateNamespaceFromControlPlaneName(name)
	err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 15*time.Minute, false, func(ctx context.Context) (bool, error) {
		kubeconfigSecret, errGet = client.CoreV1().Secrets(namespace).Get(ctx,
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
	return clientcmd.Load(kubeconfigSecret.Data[key])
}

// Merge target configuration into base configuration
func merge(base, target *clientcmdapi.Config) error {
	for k, v := range target.Clusters {
		base.Clusters[k] = v
	}

	for k, v := range target.AuthInfos {
		base.AuthInfos[k] = v
	}

	for k, v := range target.Contexts {
		base.Contexts[k] = v
	}

	if !IsHostingClusterContextPreferenceSet(base) {
		SetHostingClusterContextPreference(base, nil)
	}

	// set the current context to the nex context
	base.CurrentContext = target.CurrentContext
	return nil
}

// Assign a control plane to a context
// NOTE: function names starts with 'Assign' to have the freedom of WHERE to
// set control plane information. As of now, it is locally under 'contexts' but
// it can be set globally in the feature. We abstract that from the end-user
func AssignControlPlaneToContext(config *clientcmdapi.Config, cpName string, ctxName string) error {
	if ctx, ok := config.Contexts[ctxName]; ok {
		ctx.Namespace = ""
		ctx.Extensions = map[string]runtime.Object{
			ExtensionKubeflexKey: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					// Name: ExtensionControlPlaneName,
					// Labels: map[string]string{
					// 	ExtensionLabelManageByKubeflex: "true",
					// },
					CreationTimestamp: metav1.NewTime(time.Now()),
				},
				Data: map[string]string{
					ExtensionControlPlaneName:   cpName,
					ExtensionInitialContextName: ctxName,
				},
			},
		}
		return nil
	}
	return fmt.Errorf("error context %s does not exist in config", ctxName)
}

// Delete cluster, user and context of a given control plane
// DISCUSSION: should we restrict the usage of `kflex ctx`
// to ONLY controlplane managed by kflex ??
// It will make sense to guard any context/cluster/user that has
// nothing to do with a kflex control plane. If we do not restrict,
// then we should highly change the codeflow.
func DeleteAll(config *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	clusterName := certs.GenerateClusterName(cpName)
	authName := certs.GenerateAuthInfoAdminName(cpName)

	_, ok := config.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found for control plane %s", ctxName, cpName)
	}
	delete(config.Contexts, ctxName)
	delete(config.Clusters, clusterName)
	delete(config.AuthInfos, authName)
	return nil
}

// Get current context
func GetCurrentContext(kubeconfig string) (string, error) {
	config, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return "", err
	}
	return config.CurrentContext, nil
}

// Get hosting cluster context value set in extensions
func GetHostingClusterContext(config *clientcmdapi.Config) (string, error) {
	cm, err := unMarshallCM(config.Preferences.Extensions[ExtensionConfigName])
	if err != nil {
		return "", fmt.Errorf("error unmarshaling config map %s", err)
	}

	contextData, ok := cm.Data[ExtensionInitialContextName]
	if !ok {
		return "", fmt.Errorf("hosting cluster preference context data not set")
	}

	// make sure that context set in extension is a valid context
	_, ok = config.Contexts[contextData]
	if !ok {
		return "", fmt.Errorf("hosting cluster preference context data is set to a non-existing context")
	}

	return contextData, nil
}

func IsHostingClusterContextPreferenceSet(config *clientcmdapi.Config) bool {
	if config.Preferences.Extensions != nil {
		_, ok := config.Preferences.Extensions[ExtensionConfigName]
		if ok {
			return true
		}
	}
	return false
}

// List all contexts
func ListContexts(kubeconfig string) ([]string, error) {
	config, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	contexts := make([]string, 0, len(config.Contexts))
	for ctxName := range config.Contexts {
		contexts = append(contexts, ctxName)
	}
	return contexts, nil
}

// Load kubeconfig file (client) and merge it with control plane kubeconfig (server-side)
// CHANGES: do not write in kubeconfig anymore, instead return updated config
func LoadAndMergeClientServerKubeconfig(ctx context.Context, kubeconfig string, client kubernetes.Clientset, name string, controlPlaneType string) (*clientcmdapi.Config, error) {
	kconf, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	if controlPlaneType != string(tenancyv1alpha1.ControlPlaneTypeHost) {
		// Updates kconf
		err = LoadServerKubeconfigAndMergeIn(ctx, kconf, client, name, controlPlaneType)
	}
	return kconf, err
}

// Load control plane config (server-side) and merge it in the provided kconf
func LoadServerKubeconfigAndMergeIn(ctx context.Context, kconf *clientcmdapi.Config, client kubernetes.Clientset, name string, controlPlaneType string) error {
	cpKconf, err := loadKubeconfigFromControlPlane(ctx, client, name, controlPlaneType)
	if err != nil {
		return err
	}
	adjustConfigKeys(cpKconf, name, controlPlaneType)
	err = merge(kconf, cpKconf)
	if err != nil {
		return err
	}
	return nil
}

// Load config from provided kubeconfig file
func LoadKubeconfig(kubeconfig string) (*clientcmdapi.Config, error) {
	if kubeconfig == "" {
		kubeconfig = clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	}
	return clientcmd.LoadFromFile(kubeconfig)
}

// Rename either a cluster name, user name or context name within Kubeconfig
func RenameKey(m interface{}, oldKey string, newKey string) {
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
	}
}

// DISCUSSION: shouldn't we keep our functions as much low-level as possible?
// Rather than having SaveHostingClusterContextPreference as a function
// Shouldn't we use SetHostingClusterContextPreference and WriteKubeconfig
// whenever it is required? It seem clearer to only have a single WRITE function
// instead of SAVE function that embeds WRITE... (personal observation)
func SaveHostingClusterContextPreference(kubeconfig string) error {
	// TODO replace context parameter
	kconfig, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("setHostingClusterContextPreference: error loading kubeconfig %s", err)
	}
	SetHostingClusterContextPreference(kconfig, nil)
	// TODO replace context parameter
	return WriteKubeconfig(kubeconfig, kconfig)
}

// sets hosting cluster context to current context if userSuppliedContext is nil, otherwise set to userSuppliedContext
func SetHostingClusterContextPreference(config *clientcmdapi.Config, userSuppliedContext *string) {
	hostingContext := config.CurrentContext
	if userSuppliedContext != nil {
		hostingContext = *userSuppliedContext
	}
	runtimeObjects := make(map[string]runtime.Object)
	runtimeObjects[ExtensionConfigName] = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: ExtensionConfigName,
		},
		Data: map[string]string{
			ExtensionInitialContextName: hostingContext,
		},
	}

	config.Preferences = clientcmdapi.Preferences{
		Extensions: runtimeObjects,
	}
}

// Switch context
func SwitchContext(config *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	_, ok := config.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found", ctxName)
	}
	config.CurrentContext = ctxName
	return nil
}

// Switch to hosting cluster context
func SwitchToHostingClusterContext(config *clientcmdapi.Config, removeExtension bool) error {
	if !IsHostingClusterContextPreferenceSet(config) {
		return fmt.Errorf("hosting cluster preference context not set")
	}

	// found that the only way to unmarshal the runtime.Object into a ConfigMap
	// was to use the unMarshallCM() function based on json marshal/unmarshal
	hostingClusterContextName, err := GetHostingClusterContext(config)
	if err != nil {
		return err
	}
	config.CurrentContext = hostingClusterContextName

	// remove the extensions
	if removeExtension {
		delete(config.Preferences.Extensions, ExtensionConfigName)
	}
	return nil
}

// Write config into provided kubeconfig file
func WriteKubeconfig(kubeconfig string, config *clientcmdapi.Config) error {
	if kubeconfig == "" {
		kubeconfig = clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	}
	return clientcmd.WriteToFile(*config, kubeconfig)
}

// Watch for secret creation
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
		&corev1.Secret{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				secret := obj.(*corev1.Secret)
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

// Wait for namespace to be ready
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

			if ns.Status.Phase == corev1.NamespaceActive {
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
