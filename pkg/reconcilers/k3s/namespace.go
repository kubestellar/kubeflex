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
	"fmt"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const SystemNamespaceSuffix = "-system"
const ServerSystemNamespace = "k3s" + SystemNamespaceSuffix

// K3s service
type Namespace struct {
	*shared.BaseReconciler
}

// Generate system namespace name following this convention "$cpName-system"
func GenerateSystemNamespaceName(cpName string) string {
	return cpName + SystemNamespaceSuffix
}

// Init system namespace based on $cpName
func NewSystemNamespace(cpName string) (_ *v1.Namespace, err error) {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateSystemNamespaceName(cpName),
		},
	}, nil
}

// Reconcile namespace
// implements ControlPlaneReconciler
// NOTE: k3s controlplane belongs to the category "Single-binary" therefore
// namespace created follows "$cpName-system" naming convention
func (ns *Namespace) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	// NewSystemNamespace(hcp.Name)
	log := clog.FromContext(ctx)
	namespace, _ := NewSystemNamespace(hcp.Name)
	log.Info("k3s:core.go:Reconcile:call Reconcile namespace to create", "namespace", namespace)
	err := ns.Client.Get(ctx, client.ObjectKeyFromObject(namespace), namespace, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s:core.go:Reconcile:is not found error")
			if err := controllerutil.SetControllerReference(hcp, namespace, ns.Scheme); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			log.Info("k3s:core.go:Reconcile:client.Create on", "namespace", namespace)
			if err = ns.Client.Create(ctx, namespace, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return ctrl.Result{}, err
				}
				log.Error(err, "k3s:core.go:Reconcile:client.Create failed")
				return ctrl.Result{}, fmt.Errorf("failed to create namespace: %w", err)
			}
		} else if util.IsTransientError(err) {
			return ctrl.Result{}, err
		} else {
			log.Error(err, "k3s:core.go:Reconcile:ns.Client.Get failed")
			return ctrl.Result{}, fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	log.Info("k3s:core.go:Reconcile:end of renconcile k3s namespace")
	return ctrl.Result{}, nil
}
