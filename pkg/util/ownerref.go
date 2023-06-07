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

package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func EnsureOwnerRef(resource client.Object, ownerRef *metav1.OwnerReference) {
	if ownerRef == nil {
		return
	}
	ownerRefs := resource.GetOwnerReferences()
	i := getOwnerRefIndex(ownerRefs, ownerRef)
	if i == -1 {
		ownerRefs = append(ownerRefs, *ownerRef)
	} else {
		ownerRefs[i] = *ownerRef
	}
	resource.SetOwnerReferences(ownerRefs)
}

func getOwnerRefIndex(list []metav1.OwnerReference, ref *metav1.OwnerReference) int {
	for i := range list {
		// NOTE: The APIVersion may have changed with a new API Version, however the UID should remain the
		// same. Use either to identify the owner reference.
		if list[i].Kind == ref.Kind && (list[i].APIVersion == ref.APIVersion || list[i].UID == ref.UID) && list[i].Name == ref.Name {
			return i
		}
	}
	return -1
}
