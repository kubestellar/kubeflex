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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControlPlaneSpec defines the desired state of ControlPlane
type ControlPlaneSpec struct {
	Backend BackendDBType `json:"backend,omitempty" validate:"required,BackendDBType"`
}

// ControlPlaneStatus defines the observed state of ControlPlane
type ControlPlaneStatus struct {
	Conditions         []ControlPlaneCondition `json:"conditions"`
	ObservedGeneration int64                   `json:"observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ControlPlane is the Schema for the controlplanes API
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

type BackendDBType string

const (
	BackendDBTypeShared    BackendDBType = "shared"
	BackendDBTypeDedicated BackendDBType = "dedicated"
)

func (b BackendDBType) Validate() error {
	switch b {
	case BackendDBTypeShared, BackendDBTypeDedicated:
		return nil
	}
	return fmt.Errorf("invalid value for BackendDBType: %s", string(b))
}

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
