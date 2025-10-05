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
	"strconv"
	"time"

	"github.com/pkg/errors"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// field owner for all server-side applies
	FieldOwner = "kubeflex.kubestellar.io"
)

// Implemented by all controlplane types for central PCH processing
type PostCreateHookReconciler interface {
	ReconcileUpdatePostCreateHook(context.Context, *tenancyv1alpha1.ControlPlane) error
}

// ControlPlaneReconciler defines Reconcile loop
// each controlplane type must implement ControlPlaneReconciler as
// internal/controller/controlplane_controller.go Reconcile acts
// as a reconciler factory according to a controlplane type
type ControlPlaneReconciler interface {
	// Reconcile is part of the main kubernetes reconciliation loop which aims to
	// move the current state of the cluster closer to the desired state.
	Reconcile(context.Context, *tenancyv1alpha1.ControlPlane) (ctrl.Result, error)
}

// BaseReconciler provide common reconcilers used by other reconcilers
type BaseReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Version       string
	ClientSet     *kubernetes.Clientset
	DynamicClient *dynamic.DynamicClient
	EventRecorder record.EventRecorder
}

type SharedConfig struct {
	ExternalPort  int
	Domain        string
	HostContainer string
	IsOpenShift   bool
	ExternalURL   string
}

// Reconcile update syncing status to success
func (r *BaseReconciler) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	return r.UpdateStatusForSyncingSuccess(ctx, hcp)
}

// UpdateStatusForSyncingError change EventRecorder and ControlPlane Status to sync failed
func (r *BaseReconciler) UpdateStatusForSyncingError(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, result ctrl.Result, err error) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if r.EventRecorder != nil {
		r.EventRecorder.Event(hcp, "Warning", "SyncFail", err.Error())
	}
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileError(err))
	if err1 := r.Status().Update(context.Background(), hcp); err1 != nil {
		log.Error(err1, "update status for syncing error failed")
		return ctrl.Result{}, errors.Wrap(err, err1.Error())
	}
	if errors.Is(err, ErrPostCreateHookNotFound) {
		// Requeue after 10 seconds, don't mark as failed
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	log.Info("update status for syncing error is done")
	return result, err
}

// UpdateStatusForSyncingSuccess change EventRecorder and ControlPlane Status to success
func (r *BaseReconciler) UpdateStatusForSyncingSuccess(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if r.EventRecorder != nil {
		r.EventRecorder.Event(hcp, "Normal", "SyncSuccess", "")
	}
	tenancyv1alpha1.EnsureCondition(hcp, tenancyv1alpha1.ConditionReconcileSuccess())
	if err := r.Status().Update(context.Background(), hcp); err != nil {
		log.Error(err, "update status for syncing success failed")
		return ctrl.Result{}, err
	}
	log.Info("update status for syncing success is done")
	return ctrl.Result{}, nil
}

func (r *BaseReconciler) GetConfig(ctx context.Context) (*SharedConfig, error) {
	cmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.SystemConfigMap,
			Namespace: util.SystemNamespace,
		},
	}
	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(cmap), cmap, &client.GetOptions{})
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(cmap.Data["externalPort"])
	if err != nil {
		return nil, err
	}
	isOpenShift, err := strconv.ParseBool(cmap.Data["isOpenShift"])
	if err != nil {
		return nil, err
	}
	return &SharedConfig{
		Domain:        cmap.Data["domain"],
		HostContainer: cmap.Data["hostContainer"],
		ExternalPort:  port,
		IsOpenShift:   isOpenShift,
	}, nil
}

// UpdateStatusWithSecretRef change hcp.Status.SecretRef
func (r *BaseReconciler) UpdateStatusWithSecretRef(hcp *tenancyv1alpha1.ControlPlane, secretName, key, inClusterKey string) {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)
	hcp.Status.SecretRef = &tenancyv1alpha1.SecretReference{
		Name:         secretName,
		Namespace:    namespace,
		Key:          key,
		InClusterKey: inClusterKey,
	}
}
