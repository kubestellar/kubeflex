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

// K3s service
type Service struct {
	*shared.BaseReconciler
}

// build labels for k3s service
func serviceLabels() map[string]string {
	labels := serverLabels()
	return labels
}

// NewHeadlessService creates a new headless service for k3s statefulset
func NewHeadlessService(cpName string) (_ *v1.Service, err error) {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HeadlessServiceName,
			Namespace: GenerateSystemNamespaceName(cpName),
			Labels:    serviceLabels(),
		},
		Spec: v1.ServiceSpec{
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: v1.ClusterIPNone,
			Ports: []v1.ServicePort{
				{
					// HTTPS :443
					Port: shared.DefaultPort,
					// NOTE: why target pot should be shared.SecurePort
					TargetPort: intstr.FromInt32(shared.SecurePort),
					// HTTPS
					Name:     string(shared.DefaultPortName),
					Protocol: v1.ProtocolTCP,
				},
			},
			// Attach service to k3s apiserver
			Selector: serverLabels(),
		},
	}, nil
}

// NewClusterIPService creates a new service for k3s ingress
// TODO: refactor as a single function to create Service with option pattern
func NewClusterIPService(cpName string) (_ *v1.Service, err error) {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: GenerateSystemNamespaceName(cpName),
			Labels:    serviceLabels(),
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceTypeClusterIP,
			Ports: []v1.ServicePort{
				{
					// HTTPS :443
					Port: shared.DefaultPort,
					// k3s apiserver listen port
					TargetPort: intstr.FromInt32(APIServerPort),
					// HTTPS
					Name:     string(shared.DefaultPortName),
					Protocol: v1.ProtocolTCP,
				},
			},
			// Attach service to k3s apiserver
			Selector: serverLabels(),
		},
	}, nil
}

// GetInClusterStaticDNSRecord fetch in cluster DNS
func GetInClusterStaticDNSRecord(namespace string) string {
	return fmt.Sprintf("https://%s.%s.svc", ServiceName, namespace)
}

// GetClusterStaticDNSRecord fetch cluster dns in kubernetes form
func GetClusterStaticDNSRecord(cfg *shared.SharedConfig) string {
	return fmt.Sprintf("%s.%s", ServiceName, cfg.Domain)
}

// GetClusterServerURI compute cluster server URI to reach the k3s server
func GetClusterServerURI(cfg *shared.SharedConfig) string {
	return fmt.Sprintf("https://%s:%d", GetClusterStaticDNSRecord(cfg), cfg.ExternalPort)
}

// Reconcile a service
// implements ControlPlaneReconciler
// TODO: to implement
func (svc *Service) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	log.Info("k3s:service.go:Reconcile:starting reconciling services...")
	// Get k3s ClusterIP service to verify its existence on cluster
	k3sService := &v1.Service{}
	err := svc.Client.Get(ctx, client.ObjectKey{Namespace: GenerateSystemNamespaceName(hcp.Name), Name: ServiceName}, k3sService)
	if err != nil {
		log.Error(err, "k3s:service.go:Reconcile:r.Client.Get clusterIP service failed")
		if apierrors.IsNotFound(err) {
			// Create k3s ClusterIP service on the cluster
			log.Error(err, "service.go:Reconcile:clusterIP service is not found")
			k3sService, _ = NewClusterIPService(hcp.Name)
			if err := controllerutil.SetControllerReference(hcp, k3sService, svc.Scheme); err != nil {
				log.Error(err, "service.go:Reconcile:failed to set k3s service as secondary resource to hcp")
				return ctrl.Result{}, nil
			}
			log.Info("service.go:Reconcile: create the missing k3s clusterIP service...")
			if err := svc.Client.Create(ctx, k3sService); err != nil {
				log.Error(err, "service.go:Reconcile:creation of a new clusterIP for k3s failed")
				return ctrl.Result{}, err
			}

		} else {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
