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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/k8s"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/ocm"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/vcluster"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const kfFinalizer = "kflex.kubestellar.org/finalizer"

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Version       string
	ClientSet     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
}

//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=controlplanes/finalizers,verbs=update
//+kubebuilder:rbac:groups=tenancy.kflex.kubestellar.org,resources=postcreatehooks,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete;services
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=endpoints/restricted,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/attach,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/portforward,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:urls=/metrics,verbs=get

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

	// finalizer logic
	if hcp.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(hcp, kfFinalizer) {
			if err := r.deleteExternalResources(ctx, hcp); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(hcp, kfFinalizer)
			err := r.Update(ctx, hcp)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(hcp, kfFinalizer) {
		controllerutil.AddFinalizer(hcp, kfFinalizer)
		err = r.Update(ctx, hcp)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// check if API server is already in a ready state
	ready, _ := util.IsAPIServerDeploymentReady(r.Client, *hcp)
	if ready {
		tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionAvailable())
	} else {
		tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionUnavailable())
	}

	// select the reconciler to use for the type of control plane
	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeK8S:
		reconciler := k8s.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient)
		return reconciler.Reconcile(ctx, hcp)
	case tenancyv1alpha1.ControlPlaneTypeOCM:
		reconciler := ocm.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient)
		return reconciler.Reconcile(ctx, hcp)
	case tenancyv1alpha1.ControlPlaneTypeVCluster:
		reconciler := vcluster.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient)
		return reconciler.Reconcile(ctx, hcp)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported control plane type: %s", hcp.Spec.Type)
	}
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

func (r *ControlPlaneReconciler) deleteExternalResources(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	// add owner reference to cluster-scoped resources associated with the control plane
	// so that the Kube GC will clean those when the CP is removed
	if err := util.SetClusterScopedOwnerRefs(r.Client, r.Scheme, hcp); err != nil {
		return err
	}

	// bypass DB cleanup when running out of cluster as there is no connectivity to the DB
	if !util.IsInCluster() {
		return nil
	}
	// select the type of delete action (for now only k8s using sharedDB)
	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeK8S:
		if err := util.DropDatabase(ctx, hcp.Name, r.Client); err != nil {
			return err
		}
	case tenancyv1alpha1.ControlPlaneTypeOCM:

	case tenancyv1alpha1.ControlPlaneTypeVCluster:

	default:
		return nil
	}
	return nil
}
