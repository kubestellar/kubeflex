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

package controller

import (
	"context"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ControlPlane object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := clog.FromContext(ctx)

	log.Info("Got ControlPlane event!")

	// Fetch the hostedControlPlane instance
	hostedControlPlane := &tenancyv1alpha1.ControlPlane{}
	err := r.Client.Get(ctx, req.NamespacedName, hostedControlPlane)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	hcp := hostedControlPlane.DeepCopy()

	// check if API server is already in a ready state
	ready, _ := util.IsAPIServerDeploymentReady(r.Client, *hcp)
	if ready {
		tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionAvailable())
	} else {
		tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionUnavailable())
	}

	if err = r.ReconcileNamespace(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	crts, err := r.ReconcileCertsSecret(ctx, hcp)
	if err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	// reconcile kubeconfig for admin
	confGen := certs.ConfigGen{CpName: hcp.Name, CpHost: hcp.Name, CpPort: SecurePort}
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

	if err = r.ReconcileAPIServerIngress(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	if err = r.ReconcileCMDeployment(ctx, hcp); err != nil {
		return r.UpdateStatusForSyncingError(hcp, err)
	}

	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tenancyv1alpha1.ControlPlane{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}

func (r *ControlPlaneReconciler) UpdateStatusForSyncingError(hcp *tenancyv1alpha1.ControlPlane, e error) (ctrl.Result, error) {
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileError(e))
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(e, err.Error())
	}
	return ctrl.Result{}, err
}

func (r *ControlPlaneReconciler) UpdateStatusForSyncingSuccess(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	_ = clog.FromContext(ctx)
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileSuccess())
	err := r.Status().Update(context.Background(), hcp)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}
