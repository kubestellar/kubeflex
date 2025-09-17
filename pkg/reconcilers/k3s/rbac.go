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
	"fmt"

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

type RBAC struct {
	*shared.BaseReconciler
	ClusterRoleScriptsConfigMap        *rbacv1.ClusterRole
	ClusterRoleBindingScriptsConfigMap *rbacv1.ClusterRoleBinding
}

// NewRBAC return RBAC
func NewRBAC(br *shared.BaseReconciler) *RBAC {
	return &RBAC{
		BaseReconciler:                     br,
		ClusterRoleScriptsConfigMap:        &rbacv1.ClusterRole{},
		ClusterRoleBindingScriptsConfigMap: &rbacv1.ClusterRoleBinding{},
	}
}

// Prepare ClusterRole and ClusterRoleBinding object and manifest for ScriptsConfigMap
func (r *RBAC) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	r.ClusterRoleScriptsConfigMap = &rbacv1.ClusterRole{
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
	}
	r.ClusterRoleBindingScriptsConfigMap = &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", ScriptsConfigMapRoleBindingName, hcp.Name),
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
				Namespace: ComputeSystemNamespaceName(hcp.Name),
			},
		},
	}
	return nil
}

// Reconcile RBAC for k3s
// implements ControlPlaneReconciler
func (r *RBAC) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	// ClusterRole
	log.Info("reconcile k3s ClusterRoleScriptsConfigMap for server")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.ClusterRoleScriptsConfigMap), r.ClusterRoleScriptsConfigMap)
	switch {
	case err == nil:
		log.Info("k3s ClusterRoleScriptsConfigMap is already created", "clusterrole", r.ClusterRoleScriptsConfigMap.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s cluster role is not found error")
		log.Info("k3s SetControllerReference on cluster role")
		// Set owner reference of the API server object
		if err := controllerutil.SetControllerReference(hcp, r.ClusterRoleScriptsConfigMap, r.Scheme); err != nil {
			log.Error(err, "k3s SetControllerReference failed")
			return ctrl.Result{}, err
		}
		// Create k3s  on cluster
		log.Info("create k3s cluster role on cluster")
		if err = r.Client.Create(ctx, r.ClusterRoleScriptsConfigMap); err != nil {
			log.Error(err, "k3s creation of cluster role failed")
			return ctrl.Result{}, err
		}
	default:
		log.Error(err, "get k3s r.ClusterRoleScriptsConfigMap binding failed")
		return ctrl.Result{}, err
	}
	// ClusterRoleBinding
	log.Info("reconcile k3s role binding for server")
	err = r.Client.Get(ctx, client.ObjectKeyFromObject(r.ClusterRoleBindingScriptsConfigMap), r.ClusterRoleBindingScriptsConfigMap)
	switch {
	case err == nil:
		log.Info("k3s clusterrolebinding is already created", "clusterrolebinding", r.ClusterRoleBindingScriptsConfigMap.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s role binding  is not found error")
		log.Info("k3s SetControllerReference on role binding ")
		// Set owner reference of the API server object
		if err := controllerutil.SetControllerReference(hcp, r.ClusterRoleBindingScriptsConfigMap, r.Scheme); err != nil {
			log.Error(err, "k3s SetControllerReference role failed")
			return ctrl.Result{}, err
		}
		// Create k3s role binding on cluster
		log.Info("create k3s role biding on cluster")
		if err = r.Client.Create(ctx, r.ClusterRoleBindingScriptsConfigMap); err != nil {
			log.Error(err, "k3s creation of role binding  failed")
			return ctrl.Result{}, err
		}
	default:
		log.Error(err, "get k3s role binding failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
