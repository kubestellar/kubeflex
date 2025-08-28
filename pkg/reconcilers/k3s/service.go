/*
Copyright 2025 The KubeStellar Authors.

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

package k3s

import (
	"context"
	"fmt"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ServiceName         = "k3s"
	HeadlessServiceName = ServiceName + "-headless"
)

type Service struct {
	*shared.BaseReconciler
	Object *v1.Service
}

// serviceLables build labels for k3s service
func serviceLabels() map[string]string {
	labels := serverLabels()
	return labels
}

// NewHeadlessService creates a new headless service for k3s statefulset
// func NewHeadlessService(cpName string) (_ *v1.Service, err error) {
// 	return &v1.Service{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      HeadlessServiceName,
// 			Namespace: ComputeSystemNamespaceName(cpName),
// 			Labels:    serviceLabels(),
// 		},
// 		Spec: v1.ServiceSpec{
// 			Type:      v1.ServiceTypeClusterIP,
// 			ClusterIP: v1.ClusterIPNone,
// 			Ports: []v1.ServicePort{
// 				{
// 					// HTTPS :443
// 					Port: shared.DefaultPort,
// 					// NOTE: why target pot should be shared.SecurePort
// 					TargetPort: intstr.FromInt32(shared.SecurePort),
// 					// HTTPS
// 					Name:     string(shared.DefaultPortName),
// 					Protocol: v1.ProtocolTCP,
// 				},
// 			},
// 			// Attach service to k3s apiserver
// 			Selector: serverLabels(),
// 		},
// 	}, nil
// }

// NewClusterIPService creates a new service for k3s ingress
func NewService(br *shared.BaseReconciler) *Service {
	return &Service{
		BaseReconciler: br,
		Object:         &v1.Service{},
	}
}

// GetInClusterStaticDNSRecord fetch in cluster DNS
func GetInClusterStaticDNSRecord(namespace string) string {
	return fmt.Sprintf("https://%s.%s.svc", ServiceName, namespace)
}

// GetClusterStaticDNSRecord fetch cluster dns in kubernetes form
func GetClusterStaticDNSRecord(cpName string, cfg *shared.SharedConfig) string {
	return fmt.Sprintf("%s.%s", cpName, cfg.Domain)
}

// GetClusterServerURI compute cluster server URI to reach the k3s server
func GetClusterServerURI(cpName string, cfg *shared.SharedConfig) string {
	return fmt.Sprintf("https://%s:%d", GetClusterStaticDNSRecord(cpName, cfg), cfg.ExternalPort)
}

// Prepare service object and its manifest
func (r *Service) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	r.Object = &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: ComputeSystemNamespaceName(hcp.Name),
			Labels:    serviceLabels(),
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{
					// HTTPS :443
					Port: shared.DefaultPort,
					// k3s apiserver listen port
					TargetPort: intstr.FromInt(APIServerPort),
					// HTTPS
					Name:     string(shared.DefaultPortName),
					Protocol: v1.ProtocolTCP,
				},
			},
			// Attach service to k3s apiserver
			Selector: serverLabels(),
		},
	}
	return nil
}

// Reconcile a service
// implements ControlPlaneReconciler
func (r *Service) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("starting reconciling k3s service.")
	// Get k3s ClusterIP service to verify its existence on cluster
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("k3s service is already created")
	case apierrors.IsNotFound(err):
		// Create k3s ClusterIP service on the cluster
		log.Error(err, "k3s service is not found")
		if err := controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "failed to set k3s service as secondary resource to hcp")
			return ctrl.Result{}, nil
		}
		if err := r.Client.Create(ctx, r.Object); err != nil {
			log.Error(err, "failed to create a service for k3s")
			return ctrl.Result{}, err
		}
		log.Info("k3s service is succesfully created")
	default:
		log.Error(err, "failed to reconcile service")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
