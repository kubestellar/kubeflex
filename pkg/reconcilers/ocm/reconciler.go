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

package ocm

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
)

// OCMReconciler reconciles a OCM ControlPlane
type OCMReconciler struct {
	*shared.BaseReconciler
}

func New(cl client.Client, scheme *runtime.Scheme) *OCMReconciler {
	return &OCMReconciler{
		BaseReconciler: &shared.BaseReconciler{
			Client: cl,
			Scheme: scheme,
		},
	}
}

func (r *OCMReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	_ = clog.FromContext(ctx)

	if err := r.BaseReconciler.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileChart(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileOCMService(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileAPIServerIngress(ctx, hcp, ServiceName); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileUpdateClusterInfoJobRole(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileUpdateClusterInfoJobRoleBinding(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.ReconcileUpdateClusterInfoJob(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}
