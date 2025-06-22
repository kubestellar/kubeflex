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

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const ServiceName = "k3s-svc"

// K3s service
type Service struct {
	*shared.BaseReconciler
}

// build labels for k3s service
func serviceLabels() map[string]string {
	labels := apiServerLabels()
	return labels
}

// Init headless service for k3s apiserver
func NewService() (_ *v1.Service, err error) {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ServiceName,
			Labels: serviceLabels(),
		},
		Spec: v1.ServiceSpec{
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: v1.ClusterIPNone,
			Ports: []v1.ServicePort{
				{
					Port: shared.DefaultPort,
					// NOTE: why target pot should be shared.SecurePort
					TargetPort: intstr.FromInt32(shared.SecurePort),
					// NOTE: why should we name our port?
					Name:     string(shared.DefaultPortName),
					Protocol: v1.ProtocolTCP,
				},
			},
			// Attach service to k3s apiserver
			Selector: apiServerLabels(),
		},
	}, nil
}

// Reconcile a service
// implements ControlPlaneReconciler
// TODO: to implement
func (svc *Service) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	return svc.BaseReconciler.Reconcile(ctx, hcp)
}

