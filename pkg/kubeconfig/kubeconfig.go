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
	"strconv"
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
	ControlPlaneTypeOCMDefault      = "multicluster-controlplane"
	ControlPlaneTypeVClusterDefault = "my-vcluster"
	ControlPlaneTypeK3sDefault      = "default"
)

// adjustConfigKeys change kubeconfig default values according to its ControlPlane type with the Kubeflex kubeconfig
// default logic which is cluster=$cpName-cluster, user=$cpName-admin, context=$cpName
func adjustConfigKeys(kconf *clientcmdapi.Config, cpName, controlPlaneType string) {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeOCM):
		RenameKey(kconf.Clusters, ControlPlaneTypeOCMDefault, certs.GenerateClusterName(cpName))
		RenameKey(kconf.AuthInfos, "user", certs.GenerateAuthInfoAdminName(cpName))
		RenameKey(kconf.Contexts, ControlPlaneTypeOCMDefault, certs.GenerateContextName(cpName))
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		RenameKey(kconf.Clusters, ControlPlaneTypeVClusterDefault, certs.GenerateClusterName(cpName))
		RenameKey(kconf.AuthInfos, ControlPlaneTypeVClusterDefault, certs.GenerateAuthInfoAdminName(cpName))
		RenameKey(kconf.Contexts, ControlPlaneTypeVClusterDefault, certs.GenerateContextName(cpName))
	case string(tenancyv1alpha1.ControlPlaneTypeK3s):
		RenameKey(kconf.Clusters, ControlPlaneTypeK3sDefault, certs.GenerateClusterName(cpName))
		RenameKey(kconf.AuthInfos, ControlPlaneTypeK3sDefault, certs.GenerateAuthInfoAdminName(cpName))
		RenameKey(kconf.Contexts, ControlPlaneTypeK3sDefault, certs.GenerateContextName(cpName))
	default:
		return
	}
	kconf.CurrentContext = certs.GenerateContextName(cpName)
	kconf.Contexts[certs.GenerateContextName(cpName)] = &clientcmdapi.Context{
		Cluster:  certs.GenerateClusterName(cpName),
		AuthInfo: certs.GenerateAuthInfoAdminName(cpName),
	}
}

// Load kubeconfig from the control plane (server-side)
func loadKubeconfigFromControlPlane(ctx context.Context, client kubernetes.Clientset, name, controlPlaneType string) (*clientcmdapi.Config, error) {
	var kubeconfigSecret *corev1.Secret
	var errGet error
	namespace := util.GenerateNamespaceFromControlPlaneName(name)
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 15*time.Minute, false, func(ctx context.Context) (bool, error) {
		kubeconfigSecret, errGet = client.CoreV1().Secrets(namespace).Get(ctx,
			util.GetKubeconfSecretNameByControlPlaneType(controlPlaneType), // TODO to replace as it introduces bug
			metav1.GetOptions{})
		if errGet != nil {
			return false, nil
		}
		for _, v := range kubeconfigSecret.Data {
			if len(v) == 0 {
				return false, nil
			}
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

	if !IsHostingClusterContextSet(base) {
		err := SetHostingClusterContext(base, nil)
		if err != nil {
			return fmt.Errorf("error on ExecuteCtx: %v", err)
		}
	}

	// set the current context to the nex context
	base.CurrentContext = target.CurrentContext
	return nil
}

// Assign a control plane to a context
// NOTE: function names starts with 'Assign' to have the freedom of WHERE to
// set control plane information. As of now, it is locally under 'contexts' but
// it can be set globally in the feature. We abstract that from the end-user
func AssignControlPlaneToContext(kconf *clientcmdapi.Config, cpName string, ctxName string) error {
	if ctx, ok := kconf.Contexts[ctxName]; ok {
		var parsed map[string]runtime.Object
		kflexConfig, err := NewKubeflexContextConfig(*kconf, ctxName)
		if err != nil {
			return fmt.Errorf("error while assigning control plane to context: %v", err)
		}
		// Step 3: Assign to the context kubeflex extension data the key ExtensionControlPlaneName const and controlplane name as value
		kflexConfig.Extensions.ControlPlaneName = cpName
		if parsed, err = kflexConfig.ParseToKubeconfigExtensions(); err != nil {
			return fmt.Errorf("error while assigning control plane to context: %v", err)
		}
		ctx.Extensions = parsed
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
func DeleteAll(kconf *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	clusterName := certs.GenerateClusterName(cpName)
	authName := certs.GenerateAuthInfoAdminName(cpName)

	_, ok := kconf.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found for control plane %s", ctxName, cpName)
	}
	delete(kconf.Contexts, ctxName)
	delete(kconf.Clusters, clusterName)
	delete(kconf.AuthInfos, authName)
	return nil
}

// Get current context
func GetCurrentContext(kubeconfig string) (string, error) {
	kconf, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return "", err
	}
	return kconf.CurrentContext, nil
}

// Get hosting cluster context value set in extensions
func GetHostingClusterContext(kconf *clientcmdapi.Config) (string, error) {
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling config map %s", err)
	}
	if kflexConfig.Extensions.HostingClusterContextName == "" {
		return "", fmt.Errorf("hosting cluster context data not set")
	}
	// make sure that context set in extension is a valid context
	ctxName := kflexConfig.Extensions.HostingClusterContextName
	ctx, ok := kconf.Contexts[ctxName]
	if !ok {
		return "", fmt.Errorf("hosting cluster context data is set to a non-existing context")
	}
	// validate referenced cluster exists and has server info
	if ctx.Cluster == "" {
		return "", fmt.Errorf("hosting cluster context '%s' does not reference a cluster", ctxName)
	}
	cluster, ok := kconf.Clusters[ctx.Cluster]
	if !ok {
		return "", fmt.Errorf("cluster '%s' referenced by context '%s' does not exist", ctx.Cluster, ctxName)
	}
	if cluster.Server == "" {
		return "", fmt.Errorf("cluster '%s' referenced by context '%s' has no server defined", ctx.Cluster, ctxName)
	}
	return kflexConfig.Extensions.HostingClusterContextName, nil
}

// Check if hosting cluster context value is set within kubeconfig
func IsHostingClusterContextSet(kconf *clientcmdapi.Config) bool {
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		return false
	}
	return kflexConfig.Extensions.HostingClusterContextName != ""
}

// List all contexts
func ListContexts(kubeconfig string) ([]string, error) {
	kconf, err := LoadKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	contexts := make([]string, 0, len(kconf.Contexts))
	for ctxName := range kconf.Contexts {
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
	if err = merge(kconf, cpKconf); err != nil {
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

// Sets hosting cluster context to current context if ctxName is nil, otherwise set to ctxName
func SetHostingClusterContext(kconf *clientcmdapi.Config, ctxName *string) error {
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		return fmt.Errorf("error while setting hosting cluster context to extensions: %v", err)
	}
	hostingContext := kconf.CurrentContext
	if ctxName != nil {
		hostingContext = *ctxName
	}
	kflexContextConfig, err := NewKubeflexContextConfig(*kconf, hostingContext)
	if err != nil {
		return fmt.Errorf("error while setting hosting cluster context to context extensions: %v", err)
	}
	// Setting hosting cluster context values
	kflexConfig.Extensions.HostingClusterContextName = hostingContext
	kflexContextConfig.Extensions.IsHostingClusterContext = strconv.FormatBool(true)
	// Updating kubeconfig extensions with values
	kconf.Extensions, err = kflexConfig.ParseToKubeconfigExtensions()
	if err != nil {
		return fmt.Errorf("error while setting hosting cluster context to extensions: %v", err)
	}
	kconf.Contexts[hostingContext].Extensions, err = kflexContextConfig.ParseToKubeconfigExtensions()
	if err != nil {
		return fmt.Errorf("error while setting hosting cluster context to context extensions: %v", err)
	}
	return nil
}

// TODO: the signature is confusing. It switches context but expect controlPlane name as parameter
// NOTE: Perhaps SwitchContext should only switch context to a context name
// NOTE: Create a new function SwitchToControlPlaneContext to find a context using control plane name
// Switch context
func SwitchContext(kconf *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	_, ok := kconf.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found", ctxName)
	}
	kconf.CurrentContext = ctxName
	return nil
}

// Switch to hosting cluster context
func SwitchToHostingClusterContext(kconf *clientcmdapi.Config) error {
	hostingClusterContextName, err := GetHostingClusterContext(kconf)
	if err != nil {
		return fmt.Errorf("error while switching context to hosting cluster: %v", err)
	}
	kconf.CurrentContext = hostingClusterContextName
	return nil
}

// Write config into provided kubeconfig file
func WriteKubeconfig(kubeconfig string, kconf *clientcmdapi.Config) error {
	if kubeconfig == "" {
		kubeconfig = clientcmd.NewDefaultPathOptions().GetDefaultFilename()
	}
	return clientcmd.WriteToFile(*kconf, kubeconfig)
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

// IsContextManagedByKubeflex checks if a context is managed by KubeFlex
// by checking if it has the kubeflex extension
func IsContextManagedByKubeflex(kconf *clientcmdapi.Config, ctxName string) bool {
	ctx, exists := kconf.Contexts[ctxName]
	if !exists {
		return false
	}
	_, hasKubeflexExtension := ctx.Extensions[ExtensionKubeflexKey]
	return hasKubeflexExtension
}
