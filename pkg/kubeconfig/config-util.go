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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kubestellar/kubeflex/pkg/certs"
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

	if !IsHostingClusterContextPreferenceSet(existing) {
		SetHostingClusterContextPreference(existing, nil)
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

	delete(config.Clusters, clusterName)

	delete(config.AuthInfos, authName)

	return nil
}

func SwitchToHostingClusterContext(config *clientcmdapi.Config, removeExtension bool) error {
	if !IsHostingClusterContextPreferenceSet(config) {
		return fmt.Errorf("hosting cluster preference context not set")
	}

	// found that the only way to unmarshal the runtime.Object into a ConfigMap
	// was to use the unMarshallCM() function based on json marshal/unmarshal
	cm, err := unMarshallCM(config.Preferences.Extensions[ConfigExtensionName])
	if err != nil {
		return fmt.Errorf("error unmarshaling config map %s", err)
	}

	contextData, ok := cm.Data[InitialContextName]
	if !ok {
		return fmt.Errorf("hosting cluster preference context data not set")
	}

	// make sure that context set in extension is a valid context
	_, ok = config.Contexts[contextData]
	if !ok {
		return fmt.Errorf("hosting cluster preference context data is set to a non-existing context")
	}

	config.CurrentContext = contextData

	// remove the extensions
	if removeExtension {
		delete(config.Preferences.Extensions, ConfigExtensionName)
	}
	return nil
}

// sets hosting cluster context to current context if userSuppliedContext is nil, otherwise set to userSuppliedContext
func SetHostingClusterContextPreference(config *clientcmdapi.Config, userSuppliedContext *string) {
	hostingContext := config.CurrentContext
	if userSuppliedContext != nil {
		hostingContext = *userSuppliedContext
	}
	runtimeObjects := make(map[string]runtime.Object)
	runtimeObjects[ConfigExtensionName] = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigExtensionName,
		},
		Data: map[string]string{
			InitialContextName: hostingContext,
		},
	}

	config.Preferences = clientcmdapi.Preferences{
		Extensions: runtimeObjects,
	}
}

func IsHostingClusterContextPreferenceSet(config *clientcmdapi.Config) bool {
	if config.Preferences.Extensions != nil {
		_, ok := config.Preferences.Extensions[ConfigExtensionName]
		if ok {
			return true
		}
	}
	return false
}

func SaveHostingClusterContextPreference(ctx context.Context) error {
	kconfig, err := LoadKubeconfig(ctx)
	if err != nil {
		return fmt.Errorf("setHostingClusterContextPreference: error loading kubeconfig %s", err)
	}
	SetHostingClusterContextPreference(kconfig, nil)
	return WriteKubeconfig(ctx, kconfig)
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
