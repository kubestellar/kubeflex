package kubeconfig

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	//k8json "k8s.io/apimachinery/pkg/util/json"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"mcc.ibm.org/kubeflex/pkg/certs"
)

const (
	ConfigExtensionName = "kflex-config-extension-name"
	InitialContextName  = "kflex-initial-ctx-name"
)

func merge(existing, new *clientcmdapi.Config) error {
	for k, v := range new.Clusters {
		existing.Clusters[k] = v
	}

	for k, v := range new.AuthInfos {
		existing.AuthInfos[k] = v
	}

	for k, v := range new.Contexts {
		existing.Contexts[k] = v
	}

	if !IsInitialConfigSet(existing) {
		saveInitialContextName(existing)
	}

	// set the current context to the nex context
	existing.CurrentContext = new.CurrentContext

	return nil
}

func SwitchContext(config *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	_, ok := config.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found", ctxName)
	}
	config.CurrentContext = ctxName
	return nil
}

func DeleteContext(config *clientcmdapi.Config, cpName string) error {
	ctxName := certs.GenerateContextName(cpName)
	clusterName := certs.GenerateClusterName(cpName)
	authName := certs.GenerateAuthInfoAdminName(cpName)

	_, ok := config.Contexts[ctxName]
	if !ok {
		return fmt.Errorf("context %s not found for control plane %s", ctxName, cpName)
	}
	delete(config.Contexts, ctxName)

	_, ok = config.Clusters[clusterName]
	if !ok {
		return fmt.Errorf("cluster %s not found for control plane %s", clusterName, cpName)
	}
	delete(config.Clusters, clusterName)

	_, ok = config.AuthInfos[authName]
	if !ok {
		return fmt.Errorf("authInfo %s not found for control plane %s", authName, cpName)
	}
	delete(config.AuthInfos, authName)

	return nil
}

func SwitchToInitialContext(config *clientcmdapi.Config, removeExtension bool) error {
	if !IsInitialConfigSet(config) {
		return nil
	}
	cm, err := unMarshallCM(config.Preferences.Extensions[ConfigExtensionName])
	if err != nil {
		return fmt.Errorf("error unmarshaling config map %s", err)
	}

	contextData, ok := cm.Data[InitialContextName]
	if !ok {
		return fmt.Errorf("initial context data not set")
	}
	config.CurrentContext = contextData

	// remove the extensions
	if removeExtension {
		delete(config.Preferences.Extensions, ConfigExtensionName)
	}
	return nil
}

func saveInitialContextName(config *clientcmdapi.Config) {
	runtimeObjects := make(map[string]runtime.Object)
	runtimeObjects[ConfigExtensionName] = &corev1.ConfigMap{
		// TypeMeta: metav1.TypeMeta{
		// 	Kind:       "ConfigMap",
		// 	APIVersion: "v1",
		// },
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigExtensionName,
		},
		Data: map[string]string{
			InitialContextName: config.CurrentContext,
		},
	}

	config.Preferences = clientcmdapi.Preferences{
		Extensions: runtimeObjects,
	}
}

func IsInitialConfigSet(config *clientcmdapi.Config) bool {
	if config.Preferences.Extensions != nil {
		_, ok := config.Preferences.Extensions[ConfigExtensionName]
		if ok {
			return true
		}
	}
	return false
}

func unMarshallCM(obj runtime.Object) (*corev1.ConfigMap, error) {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshaling object %s", err)
	}
	cm := corev1.ConfigMap{}
	json.Unmarshal(jsonData, &cm)
	return &cm, nil
}
