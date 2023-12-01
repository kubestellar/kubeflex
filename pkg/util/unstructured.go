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
	"bytes"
	"fmt"
	"strings"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// Convert GroupVersionKind to GroupVersionResource
func GroupVersionKindToResource(clientset *kubernetes.Clientset, gvk schema.GroupVersionKind) (*schema.GroupVersionResource, error) {
	resourceList, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	for _, resource := range resourceList {
		for _, apiResource := range resource.APIResources {
			if apiResource.Kind == gvk.Kind && resource.GroupVersion == gvk.GroupVersion().String() {
				return &schema.GroupVersionResource{Group: gvk.Group, Version: gvk.Version, Resource: apiResource.Name}, nil
			}
		}
	}

	return nil, fmt.Errorf("GroupVersionResource not found for GroupVersionKind: %v", gvk)
}

func ToUnstructured(raw []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	err := obj.UnmarshalJSON(raw)
	if err != nil {
		fmt.Printf("Error while decoding RawExtension: %v", err)
		return nil, err
	}
	return obj, nil
}

func GetGroupVersionKindFromObject(obj *unstructured.Unstructured) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   obj.GetObjectKind().GroupVersionKind().Group,
		Version: obj.GetObjectKind().GroupVersionKind().Version,
		Kind:    obj.GetObjectKind().GroupVersionKind().Kind,
	}
}

func RenderYAML(yamlTemplate []byte, data interface{}) ([]byte, error) {
	tmpl, err := template.New("yamlTemplate").Parse(string(yamlTemplate))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// Used for generating a single string unique representation of the object for logging info
func GenerateObjectInfoString(obj unstructured.Unstructured) string {
	group := obj.GetObjectKind().GroupVersionKind().Group
	kind := strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind)

	prefix := kind
	if group != "" {
		prefix = fmt.Sprintf("%s.%s", kind, group)

	}

	return fmt.Sprintf("[%s] %s/%s", obj.GetNamespace(), prefix, obj.GetName())
}

func IsClusterScoped(gvk schema.GroupVersionKind, apiResourceLists []*metav1.APIResourceList) (bool, error) {
	for _, resourceList := range apiResourceLists {
		if resourceList.GroupVersion == gvk.Group+"/"+gvk.Version {
			for _, apiResource := range resourceList.APIResources {
				if apiResource.Kind == gvk.Kind {
					if apiResource.Namespaced {
						return false, nil
					} else {
						return true, nil
					}
				}
			}
		}
	}
	return false, fmt.Errorf("resource %s in group %s with version %s was not found", gvk.Kind, gvk.Group, gvk.Version)
}
