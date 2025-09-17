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
	// type refers to the control plane type
	// +required
	Type ControlPlaneType `json:"type,omitempty"`
	// backend refers to the database type used by the control plane
	// +required
	Backend BackendDBType `json:"backend,omitempty"`
	// bootstrapSecretRef contains a reference to the kubeconfig used to bootstrap adoption of
	// an external cluster
	// +optional
	BootstrapSecretRef *BootstrapSecretReference `json:"bootstrapSecretRef,omitempty"`
	// tokenExpirationSeconds is the expiration time for generated auth token
	// +optional
	// +kubebuilder:default:=31536000
	TokenExpirationSeconds *int64 `json:"tokenExpirationSeconds,omitempty"`
	// Deprecated: Use PostCreateHooks instead
	PostCreateHook *string `json:"postCreateHook,omitempty"`
	// Deprecated: Use PostCreateHooks instead
	PostCreateHookVars map[string]string `json:"postCreateHookVars,omitempty"`
	// PostCreateHooks specifies multiple post-creation hooks to execute
	PostCreateHooks []PostCreateHookUse `json:"postCreateHooks,omitempty"`
	// GlobalVars defines shared variables for all post-creation hooks
	// +optional
	GlobalVars map[string]string `json:"globalVars,omitempty"`
	// WaitForPostCreateHooks determines if the control plane should wait for all
	// post create hook resources to be ready before marking the control plane as ready
	// +optional
	// +kubebuilder:default:=false
	WaitForPostCreateHooks *bool `json:"waitForPostCreateHooks,omitempty"`
}

type PostCreateHookUse struct {
	// Name of the PostCreateHook resource to execute
	HookName *string `json:"hookName"`
	// Variables to pass to the hook template
	// +optional
	Vars map[string]string `json:"vars,omitempty"`
}

// ControlPlaneStatus defines the observed state of ControlPlane
type ControlPlaneStatus struct {
	Conditions         []ControlPlaneCondition `json:"conditions"`
	ObservedGeneration int64                   `json:"observedGeneration"`
	// SecretRef contains a referece to the secret containing the Kubeconfig for the control plane
	SecretRef *SecretReference `json:"secretRef,omitempty"`
	// +optional
	PostCreateHooks map[string]bool `json:"postCreateHooks,omitempty"`
	// +optional
	PostCreateHookCompleted bool `json:"postCreateHookCompleted,omitempty"`
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

// +kubebuilder:validation:Enum=k8s;ocm;vcluster;host;external;k3s
type ControlPlaneType string

const (
	ControlPlaneTypeK8S      ControlPlaneType = "k8s"
	ControlPlaneTypeOCM      ControlPlaneType = "ocm"
	ControlPlaneTypeVCluster ControlPlaneType = "vcluster"
	ControlPlaneTypeHost     ControlPlaneType = "host"
	ControlPlaneTypeExternal ControlPlaneType = "external"
	ControlPlaneTypeK3s      ControlPlaneType = "k3s"
)

// SecretReference is a reference to a secret that holds the kubeconfigs for
// a control plane hosted in the kubeflex hosting cluster, or for a kubeconfig
// for the hosting cluster itself (in the case of a control plane of type 'host') or
// for a control plane representing an external cluster.
// The 'Key' field references the kubeconfig that can be used for acccess to a control
// plane API server from outside the KubeFlex hosting cluster, while the 'InClusterKey'
// references the kubeconfig that can be used for acccess to a control
// plane API server from inside the cluster.
// We do not use ObjectReference as its use is discouraged in favor of a locally defined type.
// See ObjectReference in https://github.com/kubernetes/api/blob/master/core/v1/types.go
type SecretReference struct {
	// `namespace` is the namespace of the secret.
	// Required
	Namespace string `json:"namespace"`
	// `name` is the name of the secret.
	// Required
	Name string `json:"name"`
	// This field is present for control planes of type `k8s`, `vcluster`, `ocm`, `host`.`
	// it is not present for control planes of type `external`.
	// Controllers for control planes of type `external` should always use the `InClusterKey`.
	// +optional
	Key string `json:"key"`
	// Required
	InClusterKey string `json:"inClusterKey"`
}

// BootstrapSecretReference is a reference to a secret that holds the Kubeconfig for
// an external cluster to adopt. See SecretReference comments for why this is not
// using an ObjectReference.
type BootstrapSecretReference struct {
	// `namespace` is the namespace of the secret.
	// Required
	Namespace string `json:"namespace"`
	// `name` is the name of the secret.
	// Required
	Name string `json:"name"`
	// Required
	InClusterKey string `json:"inClusterKey"`
}

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
