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
	"context"
	_ "embed"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	// "github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ScriptsConfigMapRoleName        = ScriptsConfigMapName + "-role"
	ScriptsConfigMapRoleBindingName = ScriptsConfigMapName + "-rb"
)

// NewRole create manifest for k3s-scripts to patch k3s-config secret
func NewRole(namespace string) (*rbacv1.ClusterRole, error) {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ScriptsConfigMapRoleName,
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
func NewRoleBinding(namespace string) (*rbacv1.ClusterRoleBinding, error) {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ScriptsConfigMapRoleBindingName,
			// 			Namespace: v1.NamespaceDefault,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     ScriptsConfigMapRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      v1.NamespaceDefault,
				Namespace: namespace,
				// 		Namespace: namespace,
			},
		},
	}, nil
}

type RBAC struct {
	*shared.BaseReconciler
}

func (r *RBAC) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	log.Info("reconcile k3s role for server")
	role, _ := NewRole(GenerateSystemNamespaceName(hcp.Name))
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(role), role)
	if err != nil {
		log.Error(err, "get k3s role  failed")
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s role is not found error")
			log.Info("k3s SetControllerReference on role ")
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, role, r.Scheme); err != nil {
				log.Error(err, "k3s SetControllerReference role failed")
				return ctrl.Result{}, err
			}
			// Create k3s  on cluster
			log.Info("create k3s role on cluster")
			if err = r.Client.Create(ctx, role); err != nil {
				log.Error(err, "k3s creation of role  failed")
				return ctrl.Result{RequeueAfter: 10}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}
	log.Info("reconcile k3s role binding for server")
	roleBinding, _ := NewRoleBinding(GenerateSystemNamespaceName(hcp.Name))
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(roleBinding), roleBinding)
	if err != nil {
		log.Error(err, "get k3s role binding failed")
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s role binding  is not found error")
			log.Info("k3s SetControllerReference on role binding ")
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, roleBinding, r.Scheme); err != nil {
				log.Error(err, "k3s SetControllerReference role failed")
				return ctrl.Result{}, err
			}
			// Create k3s role binding on cluster
			log.Info("create k3s role biding on cluster")
			if err = r.Client.Create(ctx, roleBinding); err != nil {
				log.Error(err, "k3s creation of role binding  failed")
				return ctrl.Result{RequeueAfter: 10}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
