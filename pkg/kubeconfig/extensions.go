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
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	ExtensionHostingClusterContextName = "hosting-cluster-ctx-name" // BREAKING CHANGE
	ExtensionContextsIsHostingCluster  = "is-hosting-cluster-ctx"   // BREAKING CHANGE
	ExtensionInitialContextName        = "first-ctx-name"
	ExtensionControlPlaneName          = "controlplane-name"
	ExtensionKubeflexKey               = "kubeflex"
	TypeExtensionDefault               = "extensions"
	TypeExtensionLegacy                = "preferences[].extensions"
	DiagnosisStatusCritical            = "critical"
	DiagnosisStatusWarning             = "warning"
	DiagnosisStatusOK                  = "ok"
)

// Internal structure of Kubeflex global extension in a Kubeconfig file
type KubeflexExtensions struct {
	// BREAKING CHANGE
	HostingClusterContextName string `json:"hosting-cluster-ctx-name,omitempty"`
}

func (kflexExtensions KubeflexExtensions) String() string {
	return fmt.Sprintf("KubeflexExtensions: HostingClusterContextName=%s;", kflexExtensions.HostingClusterContextName)
}

// Internal structure of Kubeflex extension local to a context in a Kubeconfig file
type KubeflexContextExtensions struct {
	InitialContextName      string `json:"first-ctx-name,omitempty"`
	ControlPlaneName        string `json:"controlplane-name,omitempty"`
	IsHostingClusterContext string `json:"is-hosting-cluster-ctx,omitempty"`
}

func (kflexContextExtensions KubeflexContextExtensions) String() string {
	return fmt.Sprintf("KubeflexContextExtensions: InitialContextName=%s; ControlPlaneName=%s; IsHostingClusterContext=%s;", kflexContextExtensions.InitialContextName, kflexContextExtensions.ControlPlaneName, kflexContextExtensions.IsHostingClusterContext)
}

type RuntimeKubeflexExtension = corev1.ConfigMap
type RuntimeKubeflexExtensionData = map[string]string

func NewRuntimeKubeflexExtension() *RuntimeKubeflexExtension {
	r := &RuntimeKubeflexExtension{}
	r.ObjectMeta = metav1.ObjectMeta{
		Name:              ExtensionKubeflexKey,
		CreationTimestamp: metav1.NewTime(time.Now()),
		Namespace:         "",
	}
	r.Data = make(RuntimeKubeflexExtensionData)
	return r
}

type KubeflexConfiger interface {
	ConvertExtensionsToRuntimeExtension(receiver *RuntimeKubeflexExtension) error
	ConvertRuntimeExtensionToExtensions(producer *RuntimeKubeflexExtension) error
	ParseToKubeconfigExtensions() (map[string]runtime.Object, error)
}

type kubeflexConfig[T KubeflexExtensions | KubeflexContextExtensions] struct {
	Extensions *T
	kconf      clientcmdapi.Config
}

func newKflexConfig[T KubeflexExtensions | KubeflexContextExtensions](kconf clientcmdapi.Config) *kubeflexConfig[T] {
	return &kubeflexConfig[T]{Extensions: new(T), kconf: kconf}
}

func (kflexConfig *kubeflexConfig[T]) ConvertExtensionsToRuntimeExtension(receiver *RuntimeKubeflexExtension) error {
	// Convert to JSON
	dataJson, err := json.Marshal(kflexConfig.Extensions)
	if err != nil {
		return fmt.Errorf("json marshal of kflex config extensions failed: %v", err)
	}
	// Convert to RuntimeObject
	if err = json.Unmarshal(dataJson, &receiver.Data); err != nil {
		return fmt.Errorf("json unmarshal of kflex config extensions failed: %v", err)
	}
	return nil
}

func (kflexConfig *kubeflexConfig[T]) ConvertRuntimeExtensionToExtensions(producer *RuntimeKubeflexExtension) error {
	// Convert to JSON
	dataJson, err := json.Marshal(producer.Data)
	if err != nil {
		return fmt.Errorf("json marshal of producer data failed: %v", err)
	}
	// Convert to KubeflexExtensions
	if err = json.Unmarshal(dataJson, kflexConfig.Extensions); err != nil {
		return fmt.Errorf("json unmarshal of producer data failed: %v", err)
	}
	return nil
}

func (kflexConfig *kubeflexConfig[T]) ParseToKubeconfigExtensions() (map[string]runtime.Object, error) {
	r := NewRuntimeKubeflexExtension()
	err := kflexConfig.ConvertExtensionsToRuntimeExtension(r)
	if err != nil {
		return nil, fmt.Errorf("error while parsing kubeflex to kubeconfig extensions: %v", err)
	}
	return map[string]runtime.Object{ExtensionKubeflexKey: r}, err
}

type KubeflexConfig struct {
	*kubeflexConfig[KubeflexExtensions]
}

func NewKubeflexConfig(kconf clientcmdapi.Config) (*KubeflexConfig, error) {
	kflexConfig := newKflexConfig[KubeflexExtensions](kconf)
	if runtimeObj, ok := kconf.Extensions[ExtensionKubeflexKey]; ok {
		// Load existent kubeflex config
		runtimeExtension := &RuntimeKubeflexExtension{}
		if err := ConvertRuntimeObjectToRuntimeExtension(runtimeObj, runtimeExtension); err != nil {
			return nil, err
		}
		if err := kflexConfig.ConvertRuntimeExtensionToExtensions(runtimeExtension); err != nil {
			return nil, err
		}
	}
	return &KubeflexConfig{kubeflexConfig: kflexConfig}, nil
}

type KubeflexContextConfig struct {
	*kubeflexConfig[KubeflexContextExtensions]
	ContextName string
}

// New kubeflex config local to a context in a kubeconfig
func NewKubeflexContextConfig(kconf clientcmdapi.Config, ctxName string) (*KubeflexContextConfig, error) {
	kflexConfig := newKflexConfig[KubeflexContextExtensions](kconf)
	ctx, hasCtx := kconf.Contexts[ctxName]
	if !hasCtx {
		return nil, fmt.Errorf("context '%s' must exist within kubeconfig", ctxName)
	}
	if runtimeObj, ok := ctx.Extensions[ExtensionKubeflexKey]; ok {
		runtimeExtension := &RuntimeKubeflexExtension{}
		if err := ConvertRuntimeObjectToRuntimeExtension(runtimeObj, runtimeExtension); err != nil {
			return nil, err
		}
		if err := kflexConfig.ConvertRuntimeExtensionToExtensions(runtimeExtension); err != nil {
			return nil, err
		}
	}
	return &KubeflexContextConfig{kubeflexConfig: kflexConfig, ContextName: ctxName}, nil
}

// Unmarshal runtime.Object into RuntimeKubeflexExtension
func ConvertRuntimeObjectToRuntimeExtension(data runtime.Object, receiver *RuntimeKubeflexExtension) error {
	dataJson, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(dataJson, &receiver)
	if err != nil {
		return err
	}
	return nil
}

// CheckGlobalKubeflexExtension checks the status of the global kubeflex extension
func CheckGlobalKubeflexExtension(kconf clientcmdapi.Config) (string, *KubeflexExtensions) {
	runtimeObj, exists := kconf.Extensions[ExtensionKubeflexKey]
	if !exists {
		return DiagnosisStatusCritical, nil
	}

	runtimeExtension := &RuntimeKubeflexExtension{}
	if err := ConvertRuntimeObjectToRuntimeExtension(runtimeObj, runtimeExtension); err != nil {
		return DiagnosisStatusCritical, nil
	}

	// Check if the extension has any data
	if len(runtimeExtension.Data) == 0 {
		return DiagnosisStatusWarning, nil
	}

	// Parse the data into KubeflexExtensions
	kflexConfig := newKflexConfig[KubeflexExtensions](kconf)
	if err := kflexConfig.ConvertRuntimeExtensionToExtensions(runtimeExtension); err != nil {
		return DiagnosisStatusCritical, nil
	}

	return DiagnosisStatusOK, kflexConfig.Extensions
}
