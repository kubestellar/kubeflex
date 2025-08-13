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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	SystemNamespaceSuffix = "-system"
	ServerSystemNamespace = "k3s" + SystemNamespaceSuffix
)

// Namespace defines tenant namespace
type Namespace struct {
	*shared.BaseReconciler
	Object *v1.Namespace
}

// GenerateSystemNamespaceName follows this convention "$cpName-system"
func GenerateSystemNamespaceName(cpName string) string {
	return cpName + SystemNamespaceSuffix
}

// NewSystemNamespace Init system namespace based on $cpName
// namespace created follows "$cpName-system" naming convention
func NewSystemNamespace(cpName string) (_ *v1.Namespace, err error) {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateSystemNamespaceName(cpName),
		},
	}, nil
}

// Reconcile namespace
// implements ControlPlaneReconciler
func (ns *Namespace) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	ns.Object, _ = NewSystemNamespace(hcp.Name)
	log.Info("call Reconcile namespace to create", "namespace", ns.Object)
	err := ns.Client.Get(ctx, client.ObjectKeyFromObject(ns.Object), ns.Object, &client.GetOptions{})
	switch {
	case err == nil:
		log.Info("namespace is already created", "namespace", ns.Object.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "is not found error")
		if err := controllerutil.SetControllerReference(hcp, ns.Object, ns.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to SetControllerReference: %w", err)
		}
		log.Info("client.Create on", "namespace", ns.Object)
		if err = ns.Client.Create(ctx, ns.Object, &client.CreateOptions{}); err != nil {
			log.Error(err, "client.Create failed")
			return ctrl.Result{}, fmt.Errorf("failed to create namespace: %w", err)
		}
	default:
		log.Error(err, "reconcile namespace failed")
	}
	log.Info("end of renconcile k3s namespace")
	return ctrl.Result{}, nil
}
