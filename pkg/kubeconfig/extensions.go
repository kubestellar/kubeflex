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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	ExtensionConfigName                = "kflex-config-extension-name" // Unchanged otherwise breaking change
	ExtensionHostingClusterContextName = "kflex-initial-ctx-name"      // Unchanged otherwise breaking change
	ExtensionInitialContextName        = "first-context-name"
	ExtensionControlPlaneName          = "controlplane-name"
	ExtensionKubeflexKey               = "kubeflex"
	ExtensionLabelManageByKubeflex     = "kubeflex.dev/is-managed"
	ControlPlaneTypeOCMDefault         = "multicluster-controlplane"
	ControlPlaneTypeVClusterDefault    = "my-vcluster"
	TypeExtensionDefault               = "extensions"
	TypeExtensionLegacy                = "preferences[].extensions"
)

// Internal structure of Kubeflex global extension in a Kubeconfig file
type KubeflexExtensions struct {
	// Must change --> BREAKING CHANGE
	ConfigName string // `json:"kflex-config-extension-name,omitempty"`
	// Should change --> BREAKING CHANGE
	HostingClusterContextName string // `json:"kflex-initial-ctx-name,omitempty"`
}

// Internal structure of Kubeflex extension local to a context in a Kubeconfig file
type KubeflexContextExtensions struct {
	InitialContextName    string // `json:"first-context-name,omitempty"`
	ControlPlaneName      string // `json:"controlplane-name,omitempty"`
	LabelManageByKubeflex string // `json:"kubeflex.dev/is-managed,omitempty"`
}

type RuntimeKubeflexExtension struct {
	corev1.ConfigMap
}

type RuntimeKubeflexExtensionData = map[string]string

func NewRuntimeKubeflexExtension() RuntimeKubeflexExtension {
	r := RuntimeKubeflexExtension{}
	r.ObjectMeta = metav1.ObjectMeta{
		Name:              ExtensionKubeflexKey,
		CreationTimestamp: v1.NewTime(time.Now()),
	}
	r.Data = make(RuntimeKubeflexExtensionData)
	return r
}

func (runtimeKflex RuntimeKubeflexExtension) SetKubeflexConfigName(v string) {
	runtimeKflex.Data[ExtensionConfigName] = v
}

func (runtimeKflex RuntimeKubeflexExtension) GetKubeflexConfigName() string {
	return runtimeKflex.Data[ExtensionConfigName]
}

func (runtimeKflex RuntimeKubeflexExtension) SetInitialContextName(v string) {
	runtimeKflex.Data[ExtensionInitialContextName] = v
}

func (runtimeKflex RuntimeKubeflexExtension) GetInitialContextName() string {
	return runtimeKflex.Data[ExtensionInitialContextName]
}

func (runtimeKflex RuntimeKubeflexExtension) SetControlPlaneName(v string) {
	runtimeKflex.Data[ExtensionControlPlaneName] = v
}

func (runtimeKflex RuntimeKubeflexExtension) GetControlPlaneName() string {
	return runtimeKflex.Data[ExtensionControlPlaneName]
}

func (runtimeKflex RuntimeKubeflexExtension) SetLabelManageByKubeflex(v string) {
	runtimeKflex.Data[ExtensionLabelManageByKubeflex] = v
}

func (runtimeKflex RuntimeKubeflexExtension) GetLabelManageByKubeflex() string {
	return runtimeKflex.Data[ExtensionLabelManageByKubeflex]
}

func (runtimeKflex RuntimeKubeflexExtension) SetKubeflexHostingClusterContextName(v string) {
	runtimeKflex.Data[ExtensionHostingClusterContextName] = v
}

func (runtimeKflex RuntimeKubeflexExtension) GetKubeflexHostingClusterContextName() string {
	return runtimeKflex.Data[ExtensionHostingClusterContextName]
}

type KubeflexExtensioner interface {
	ParseToKubeconfigExtensions() (map[string]runtime.Object, error)
	ParseToRuntimeKubefleExtensionData() (parsed RuntimeKubeflexExtensionData, err error)
}

type KubeflexConfig struct {
	kconf      clientcmdapi.Config
	Extensions KubeflexExtensions // under `extensions.kubeflex`
	// runtimeExtension RuntimeKubeflexExtension
}

func NewKubeflexConfig(kconf clientcmdapi.Config) (*KubeflexConfig, error) {
	kflexConfig := KubeflexConfig{kconf: kconf, Extensions: KubeflexExtensions{}}
	if runtimeObj, ok := kconf.Extensions[ExtensionKubeflexKey]; ok {
		runtimeExtension := RuntimeKubeflexExtension{}
		err := ConvertRuntimeObjectToRuntimeKubeflexExtension(runtimeObj, runtimeExtension)
		if err != nil {
			return nil, err
		}
		kflexConfig.Extensions.ConfigName = runtimeExtension.GetKubeflexConfigName()
		kflexConfig.Extensions.HostingClusterContextName = runtimeExtension.GetKubeflexHostingClusterContextName()
	}
	return &kflexConfig, nil
}

// func NewKubeflexContextConfig() *KubeflexContextConfig {
// 	return
// }

// Unmarshal runtime.Object into RuntimeKubeflexExtension
func ConvertRuntimeObjectToRuntimeKubeflexExtension(data runtime.Object, receiver RuntimeKubeflexExtension) error {
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

// Parse KubeflexExtensions into RuntimeKubeflexExtensionData
func (kflexConfig KubeflexConfig) ParseToRuntimeKubefleExtensionData() (parsed RuntimeKubeflexExtensionData, err error) {
	data, err := json.Marshal(kflexConfig.Extensions)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		return nil, err
	}
	return parsed, err
}

// Parse KubeflexExtensions into Kubeconfig Extensions
func (kflexConfig KubeflexConfig) ParseToKubeconfigExtensions() (map[string]runtime.Object, error) {
	runtimeExtensionData, err := kflexConfig.ParseToRuntimeKubefleExtensionData()
	if err != nil {
		return nil, err
	}
	runtimeExtension := NewRuntimeKubeflexExtension()
	runtimeExtension.Data = runtimeExtensionData
	return map[string]runtime.Object{ExtensionKubeflexKey: &runtimeExtension}, nil
}

type KubeflexContextConfig struct {
	kconf             clientcmdapi.Config
	Context           string
	ContextExtensions KubeflexContextExtensions
}
