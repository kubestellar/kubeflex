/*
Copyright 2025 The KubeStellar Authors.

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

package k3s

import (
	// "context"
	_ "embed"

	// tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	// "github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	// batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	// apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/utils/ptr"
	// ctrl "sigs.k8s.io/controller-runtime"
	// "sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	// clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ScriptsConfigMapRoleName        = ScriptsConfigMapName + "-role"
	ScriptsConfigMapRoleBindingName = ScriptsConfigMapName + "-rb"
)

// NewRole create manifest for k3s-scripts to patch k3s-config secret
func NewRole(namespace string) (*rbacv1.Role, error) {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: ScriptsConfigMapRoleName, Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					v1.GroupName,
				},
				Resources: []string{
					string(v1.ResourceSecrets),
				},
				Verbs: []string{
					"patch",
				},
			},
		},
	}, nil
}

// NewRoleBinding create manifest to bind k3s-scripts role to default service account
func NewRoleBinding(namespace string) (*rbacv1.RoleBinding, error) {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ScriptsConfigMapRoleBindingName,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     rbacv1.GroupKind,
			Name:     ScriptsConfigMapRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      v1.NamespaceDefault,
				Namespace: namespace,
			},
		},
	}, nil
}
