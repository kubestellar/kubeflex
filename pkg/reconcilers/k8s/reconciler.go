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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// K8sReconciler reconciles a k8s ControlPlane
type K8sReconciler struct {
	*shared.BaseReconciler
}

func New(cl client.Client, scheme *runtime.Scheme, version string, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, eventRecorder record.EventRecorder) *K8sReconciler {
	return &K8sReconciler{
		BaseReconciler: &shared.BaseReconciler{
			Client:        cl,
			Scheme:        scheme,
			ClientSet:     clientSet,
			DynamicClient: dynamicClient,
			EventRecorder: eventRecorder,
		},
	}
}

func (r *K8sReconciler) Reconcile(ctx context.Context, hcp *v1alpha1.ControlPlane) (ctrl.Result, error) {
	var routeURL string
	log := clog.FromContext(ctx)

	cfg, err := r.BaseReconciler.GetConfig(ctx)
	if err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err := r.BaseReconciler.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileAPIServerService(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if cfg.IsOpenShift {
		if err = r.ReconcileAPIServerRoute(ctx, hcp, "", shared.SecurePort, cfg.Domain); err != nil {
			return r.UpdateStatusForSyncingError(hcp, err)
		}
		routeURL, err = r.GetAPIServerRouteURL(ctx, hcp)
		if err != nil {
			return r.UpdateStatusForSyncingError(hcp, err)
		}
		// re-queue until valid route URL is retrieved
		if routeURL == "" {
			return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
		}
	} else {
		if err = r.ReconcileAPIServerIngress(ctx, hcp, "", shared.DefaulPort, cfg.Domain); err != nil {
			return r.UpdateStatusForSyncingError(hcp, err)
		}
	}

	crts, err := r.ReconcileCertsSecret(ctx, hcp, cfg, routeURL)
	if err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	confGen := &certs.ConfigGen{
		CpName:     hcp.Name,
		CpHost:     hcp.Name,
		CpPort:     cfg.ExternalPort,
		CpDomain:   cfg.Domain,
		CpExtraDNS: routeURL}
	// reconcile kubeconfig for admin
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

	if err = r.ReconcileAPIServerDeployment(ctx, hcp, cfg.IsOpenShift); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileCMDeployment(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	r.UpdateStatusWithSecretRef(hcp, util.AdminConfSecret, util.KubeconfigSecretKeyDefault, util.KubeconfigSecretKeyInCluster)

	// FIXED: Process hooks regardless of Ready condition to break chicken-and-egg cycle
	// The main controller will handle the Ready logic based on waitForPostCreateHooks
	if hcp.Spec.PostCreateHook != nil || len(hcp.Spec.PostCreateHooks) > 0 {
		log.Info("Processing post-create hooks", "controlPlane", hcp.Name,
			"legacyHook", hcp.Spec.PostCreateHook,
			"newHooks", len(hcp.Spec.PostCreateHooks))

		if err := r.ReconcileUpdatePostCreateHook(ctx, hcp); err != nil {
			// For transient errors, log and requeue but don't fail completely
			if util.IsTransientError(err) {
				log.Info("Transient error processing hooks, will retry", "error", err)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
			// For permanent errors, log but continue - let main controller handle final status
			log.Error(err, "Failed to process post-create hooks")
			// Don't return error here - let the main controller decide final readiness
		} else {
			log.Info("Successfully processed post-create hooks", "controlPlane", hcp.Name)
		}
	}

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}
