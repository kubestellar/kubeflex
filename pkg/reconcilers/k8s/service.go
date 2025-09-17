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

	"github.com/kubestellar/kubeflex/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
)

func (r *K8sReconciler) ReconcileAPIServerService(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// create service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(service), service, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			service := generateAPIServerService(hcp.Name, namespace)
			if err := controllerutil.SetControllerReference(hcp, service, r.Scheme); err != nil {
				return nil
			}
			err = r.Client.Create(context.TODO(), service, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func generateAPIServerService(name, namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": util.APIServerDeploymentName,
			},
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port:       shared.DefaultPort,
					TargetPort: intstr.FromInt(shared.SecurePort),
					Name:       shared.DefaultPortName,
					Protocol:   "TCP",
				},
			},
		},
	}
}
