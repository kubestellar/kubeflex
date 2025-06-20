/*
Copyright 2024 The KubeStellar Authors.

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

package external

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// ExternalReconciler reconciles an "external" ControlPlane to adopt an external cluster with the ControlPlane abstraction
type ExternalReconciler struct {
	*shared.BaseReconciler
}

func New(cl client.Client, scheme *runtime.Scheme, version string, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, eventRecorder record.EventRecorder) *ExternalReconciler {
	return &ExternalReconciler{
		BaseReconciler: &shared.BaseReconciler{
			Client:        cl,
			Scheme:        scheme,
			ClientSet:     clientSet,
			DynamicClient: dynamicClient,
			EventRecorder: eventRecorder,
		},
	}
}

func (r *ExternalReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {

	if err := r.BaseReconciler.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	if err := r.ReconcileKubeconfigFromBootstrapSecret(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	r.UpdateStatusWithSecretRef(hcp, util.AdminConfSecret, util.KubeconfigSecretKeyDefault, util.KubeconfigSecretKeyInCluster)

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}
