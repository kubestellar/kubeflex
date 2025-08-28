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

package host

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// HostReconciler reconciles a Host ControlPlane that exposed the hosting cluster with the ControlPlane abstraction
type HostReconciler struct {
	*shared.BaseReconciler
}

func New(cl client.Client, scheme *runtime.Scheme, version string, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, eventRecorder record.EventRecorder) *HostReconciler {
	return &HostReconciler{
		BaseReconciler: &shared.BaseReconciler{
			Client:        cl,
			Scheme:        scheme,
			ClientSet:     clientSet,
			DynamicClient: dynamicClient,
			EventRecorder: eventRecorder,
		},
	}
}

func (r *HostReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	_ = clog.FromContext(ctx)

	if err := r.BaseReconciler.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	if err := r.ReconcileServiceAccount(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	if err := r.ReconcileServiceAccountSecret(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	if err := r.ReconcileKubeconfigSecret(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	if err := r.ReconcileClusterRoleBinding(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
	}

	r.UpdateStatusWithSecretRef(hcp, util.AdminConfSecret, util.KubeconfigSecretKeyDefault, util.KubeconfigSecretKeyInCluster)

	if hcp.Spec.PostCreateHook != nil &&
		tenancyv1alpha1.HasConditionAvailable(hcp.Status.Conditions) {
		if err := r.ReconcileUpdatePostCreateHook(ctx, hcp); err != nil {
			return r.UpdateStatusForSyncingError(ctx, hcp, ctrl.Result{}, err)
		}
	}

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}

func (r *HostReconciler) checkOnlyOneCPOfTypeHostExists(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	list := &tenancyv1alpha1.ControlPlaneList{}
	if err := r.Client.List(ctx, list, &client.ListOptions{}); err != nil {
		return err
	}
	for _, item := range list.Items {
		if item.Spec.Type == tenancyv1alpha1.ControlPlaneTypeHost && item.GetName() != hcp.GetName() {
			return fmt.Errorf("found another control plane with name %s of type %s. Only one control plane of type %s can be created",
				item.Name, tenancyv1alpha1.ControlPlaneTypeHost, tenancyv1alpha1.ControlPlaneTypeHost)
		}

	}
	return nil
}
