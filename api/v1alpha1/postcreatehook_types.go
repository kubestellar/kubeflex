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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Var defines a name/value pair for template variables
type Var struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PostCreateHookSpec defines the desired state of PostCreateHook
type PostCreateHookSpec struct {
	Templates   []Manifest `json:"templates,omitempty"`
	DefaultVars []Var      `json:"defaultVars,omitempty"`
}

// PostCreateHookStatus defines the observed state of PostCreateHook
type PostCreateHookStatus struct {
	Conditions         []ControlPlaneCondition `json:"conditions"`
	ObservedGeneration int64                   `json:"observedGeneration"`
	// SecretRef contains a referece to the secret containing the Kubeconfig for the control plane
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// PostCreateHook is the Schema for the controlplanes API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,shortName={pch,pchs}
type PostCreateHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostCreateHookSpec   `json:"spec,omitempty"`
	Status PostCreateHookStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostCreateHookList contains a list of PostCreateHook
type PostCreateHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostCreateHook `json:"items"`
}

// Manifest represents a resource to be deployed
type Manifest struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	runtime.RawExtension `json:",inline"`
}

func init() {
	SchemeBuilder.Register(&PostCreateHook{}, &PostCreateHookList{})
}
