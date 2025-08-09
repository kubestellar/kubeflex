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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/external"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/host"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/k8s"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/ocm"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
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
	EventRecorder record.EventRecorder
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

	// Update observedGeneration if it doesn't match the current generation
	if hcp.Status.ObservedGeneration != hcp.Generation {
		hcp.Status.ObservedGeneration = hcp.Generation
	}

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

	// PHASE 1: Type-specific controlplane reconciliation
	log.Info("Phase 1: Infrastructure setup and kubeconfig creation", "type", hcp.Spec.Type)
	var reconcileResult ctrl.Result
	var reconcileError error

	var reconciler shared.ControlPlaneTypeReconciler

	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeK8S:
		reconciler = k8s.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient, r.EventRecorder)
	case tenancyv1alpha1.ControlPlaneTypeOCM:
		reconciler = ocm.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient, r.EventRecorder)
	case tenancyv1alpha1.ControlPlaneTypeVCluster:
		reconciler = vcluster.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient, r.EventRecorder)
	case tenancyv1alpha1.ControlPlaneTypeHost:
		reconciler = host.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient, r.EventRecorder)
	case tenancyv1alpha1.ControlPlaneTypeExternal:
		reconciler = external.New(r.Client, r.Scheme, r.Version, r.ClientSet, r.DynamicClient, r.EventRecorder)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported control plane type: %s", hcp.Spec.Type)
	}

	// Call the type-specific reconciler to handle infrastructure and kubeconfig setup
	reconcileResult, reconcileError = reconciler.Reconcile(ctx, hcp)
	if reconcileError != nil {
		log.Error(reconcileError, "Type-specific reconciliation failed")
		return reconcileResult, reconcileError
	}

	// Refresh the hcp object after infrastructure reconciliation
	if err := r.Get(ctx, client.ObjectKey{Name: hcp.Name}, hcp); err != nil {
		log.Error(err, "Failed to refresh ControlPlane after infrastructure reconciliation")
		return ctrl.Result{}, err
	}

	// PHASE 2: Check API server readiness
	apiServerReady, err := util.IsAPIServerDeploymentReady(log, r.Client, *hcp)
	if err != nil {
		log.Error(err, "Error checking API server readiness", "controlPlane", hcp.Name)
	}
	log.Info("API server readiness check", "controlPlane", hcp.Name, "apiServerReady", apiServerReady)

	// Requeue if API Server is Not Ready
	if !apiServerReady {
		log.Info("API Server Not Ready. Requeuing...", "controlPlane", hcp.Name)
		// Update Status
		tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionUnavailable())
		if err = r.Status().Update(ctx, hcp); err != nil {
			log.Error(err, "Failed to update ControlPlane status")
		}
		return ctrl.Result{RequeueAfter: time.Second * 15}, nil
	}

	// PHASE 3: PostCreateHook processing
	if hcp.Spec.PostCreateHook != nil || len(hcp.Spec.PostCreateHooks) > 0 {
		log.Info("Processing PostCreateHooks with complete kubeconfig")

		if err := reconciler.ReconcileUpdatePostCreateHook(ctx, hcp); err != nil {
			log.Error(err, "Failed to process PostCreateHooks")
			// Don't return error immediately - let status logic handle it
		}

		// Refresh hcp object after PCH processing
		if err := r.Get(ctx, client.ObjectKey{Name: hcp.Name}, hcp); err != nil {
			log.Error(err, "Failed to refresh ControlPlane after hook processing")
		}
	}

	// Determine overall controlplane readiness based on both API server and PCHs
	log.Info("Determining final control plane readiness")
	if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
		// NEW BEHAVIOR: CP ready = API Server ready AND PostCreateHooks completed
		log.Info("Checking both API server and PostCreateHook completion",
			"apiServerReady", apiServerReady,
			"postCreateHookCompleted", hcp.Status.PostCreateHookCompleted)

		if apiServerReady && hcp.Status.PostCreateHookCompleted {
			log.Info("Both API server and PostCreateHooks are ready, marking control plane as ready")
			tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionAvailable())
		} else {
			if !apiServerReady && !hcp.Status.PostCreateHookCompleted {
				log.Info("Waiting for both API server and post-create hooks")
				tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionWaitingForPostCreateHooks())
			} else if !apiServerReady {
				log.Info("Waiting for API server readiness")
				tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionUnavailable())
			} else {
				log.Info("Waiting for post-create hooks to complete")
				tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionWaitingForPostCreateHooks())
			}
		}
	} else {
		// DEFAULT BEHAVIOR: CP ready = API Server ready (ignore hooks)
		if apiServerReady {
			log.Info("API server ready, marking control plane as ready (not waiting for hooks)")
			tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionAvailable())
		} else {
			log.Info("API server not ready, marking control plane as unavailable")
			tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionUnavailable())
		}
	}

	// Update final status
	err = r.Status().Update(ctx, hcp)
	if err != nil {
		log.Error(err, "Failed to update ControlPlane final status")
		return ctrl.Result{}, err
	}

	// If we're still waiting for hooks to complete, requeue to check again later
	if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks &&
		(!apiServerReady || !hcp.Status.PostCreateHookCompleted) {
		log.Info("Requeuing to check completion later")
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	// Return the result from the type-specific reconciler
	return reconcileResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("kubeflex-controlplane-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&tenancyv1alpha1.ControlPlane{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Secret{}).
		Watches(&corev1.Secret{}, enqueueSecretsOfInterest(mgr.GetLogger())).
		Complete(r)
}

func enqueueSecretsOfInterest(logger logr.Logger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		secret, ok := o.(*corev1.Secret)
		if !ok {
			return nil
		}

		// Check if the secret has the specified name
		if secret.Name == util.VClusterKubeConfigSecret {
			cpName, err := util.ControlPlaneNameFromNamespace(secret.Namespace)
			if err == nil {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cpName}}}
			}
			logger.Info("Ignoring non-ControlPlane Secret named "+util.VClusterKubeConfigSecret, "namespace", secret.Namespace)
		}

		return []reconcile.Request{}
	})
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
