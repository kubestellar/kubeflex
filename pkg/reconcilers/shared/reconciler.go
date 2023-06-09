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

package shared

import (
	"context"

	"github.com/pkg/errors"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

// BaseReconciler provide common reconcilers used by other reconcilers
type BaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *BaseReconciler) UpdateStatusForSyncingError(hcp *tenancyv1alpha1.ControlPlane, e error) (ctrl.Result, error) {
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileError(e))
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(e, err.Error())
	}
	return ctrl.Result{}, err
}

func (r *BaseReconciler) UpdateStatusForSyncingSuccess(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	_ = clog.FromContext(ctx)
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileSuccess())
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}
