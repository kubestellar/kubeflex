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
)

// ControlPlaneSpec defines the desired state of ControlPlane
type ControlPlaneSpec struct {
	Type           ControlPlaneType `json:"type,omitempty"`
	Backend        BackendDBType    `json:"backend,omitempty"`
	PostCreateHook *string          `json:"postCreateHook,omitempty"`
}

// ControlPlaneStatus defines the observed state of ControlPlane
type ControlPlaneStatus struct {
	Conditions         []ControlPlaneCondition `json:"conditions"`
	ObservedGeneration int64                   `json:"observedGeneration"`
	// SecretRef contains a referece to the secret containing the Kubeconfig for the control plane
	SecretRef *SecretReference `json:"secretRef,omitempty"`
	// +optional
	PostCreateHooks map[string]bool `json:"postCreateHooks,omitempty"`
}

// ControlPlane is the Schema for the controlplanes API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,shortName={cp,cps}
type ControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneSpec   `json:"spec,omitempty"`
	Status ControlPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ControlPlaneList contains a list of ControlPlane
type ControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlane `json:"items"`
}

// +kubebuilder:validation:Enum=shared;dedicated
type BackendDBType string

const (
	BackendDBTypeShared    BackendDBType = "shared"
	BackendDBTypeDedicated BackendDBType = "dedicated"
)

// +kubebuilder:validation:Enum=k8s;ocm;vcluster
type ControlPlaneType string

const (
	ControlPlaneTypeK8S      ControlPlaneType = "k8s"
	ControlPlaneTypeOCM      ControlPlaneType = "ocm"
	ControlPlaneTypeVCluster ControlPlaneType = "vcluster"
)

// We do not use ObjectReference as its use is discouraged in favor of a locally defined type.
// See ObjectReference in https://github.com/kubernetes/api/blob/master/core/v1/types.go
type SecretReference struct {
	// `namespace` is the namespace of the secret.
	// Required
	Namespace string `json:"namespace"`
	// `name` is the name of the secret.
	// Required
	Name string `json:"name"`
}

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
