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

package k8s

import (
	"context"

	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
)

// K8sReconciler reconciles a k8s ControlPlane
type K8sReconciler struct {
	*shared.BaseReconciler
}

func New(cl client.Client, scheme *runtime.Scheme) *K8sReconciler {
	return &K8sReconciler{
		BaseReconciler: &shared.BaseReconciler{
			Client: cl,
			Scheme: scheme,
		},
	}
}

func (r *K8sReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	_ = clog.FromContext(ctx)

	if err := r.BaseReconciler.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	crts, err := r.ReconcileCertsSecret(ctx, hcp)
	if err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	// reconcile kubeconfig for admin
	confGen := certs.ConfigGen{CpName: hcp.Name, CpHost: hcp.Name, CpPort: shared.SecurePort}
	confGen.Target = certs.Admin
	if err = r.ReconcileKubeconfigSecret(ctx, crts, confGen, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	// reconcile kubeconfig for cm
	confGen.Target = certs.ControllerManager
	confGen.CpHost = hcp.Name
	if err = r.ReconcileKubeconfigSecret(ctx, crts, confGen, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileAPIServerDeployment(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileAPIServerService(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileAPIServerIngress(ctx, hcp, "", shared.SecurePort); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileCMDeployment(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}
