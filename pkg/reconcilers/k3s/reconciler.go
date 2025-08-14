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
	"time"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K3sReconciler embeds all k3s components
type K3sReconciler struct {
	*Namespace             // k3s namespace
	*Service               // k3s service
	*Server                // k3s api server
	*KubeconfigSecret      // k3s secret
	*ScriptsConfigMap      // k3s scripts configmap
	*RBAC                  // k3s rbac
	*Ingress               // k3s ingress
	*BootstrapSecretJob    // k3s job
	*shared.BaseReconciler // shared base controller
}

const RetryAfterDuration = 5 * time.Second

// New create a base reconciler
func New(cl client.Client, scheme *runtime.Scheme, version string, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, eventRecorder record.EventRecorder) *K3sReconciler {
	br := shared.BaseReconciler{
		Client:        cl,
		Scheme:        scheme,
		ClientSet:     clientSet,
		DynamicClient: dynamicClient,
		EventRecorder: eventRecorder,
	}

	return &K3sReconciler{
		BaseReconciler:     &br,
		Namespace:          NewSystemNamespace(&br),
		BootstrapSecretJob: NewBootstrapSecretJob(&br),
		Service:            NewService(&br),
		Server:             NewServer(&br),
		KubeconfigSecret:   NewKubeconfigSecret(&br),
		ScriptsConfigMap:   NewScriptsConfigMap(&br),
		Ingress:            NewIngress(&br),
		RBAC:               NewRBAC(&br),
	}
}

// Reconcile K3s control plane
// implements ControlPlaneReconciler
func (r *K3sReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	// Reconcile mandatory k3s namespace
	if result, err := r.Namespace.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s RBAC
	if result, err := r.RBAC.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s ConfigMap
	if result, err := r.ScriptsConfigMap.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s Server
	if result, err := r.Server.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s Service
	if result, err := r.Service.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s Ingress
	if result, err := r.Ingress.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s Secret
	if result, err := r.KubeconfigSecret.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Reconcile k3s Job
	if result, err := r.BootstrapSecretJob.Reconcile(ctx, hcp); err != nil {
		if util.IsTransientError(err) {
			return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
		}
		return r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, result, err)
	}
	// Update secretref status
	// NOTE perhaps a better design would be to embed each object manifest
	// within its reconciler (see r.Namespace.Object.Name)
	hcp.Status.SecretRef = &tenancyv1alpha1.SecretReference{
		Namespace:    r.Namespace.Object.Name,
		Name:         r.KubeconfigSecret.Object.Name,
		Key:          KubeconfigSecretKey,
		InClusterKey: KubeconfigSecretKeyInCluster,
	}
	// NOTE add PostCreateHook if it makes sense below

	// Update reconcile status to success
	return r.BaseReconciler.Reconcile(ctx, hcp)
}
