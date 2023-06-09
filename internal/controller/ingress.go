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

	"github.com/kubestellar/kubeflex/pkg/util"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	IngressClassNameNGINX = "nginx"
)

var (
	pathTypePrefix = networkingv1.PathTypePrefix
)

func (r *ControlPlaneReconciler) ReconcileAPIServerIngress(ctx context.Context, name string, owner *metav1.OwnerReference) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	// create service object
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(ingress), ingress, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ingress = generateAPIServerIngress(name, namespace)
			util.EnsureOwnerRef(ingress, owner)
			err = r.Client.Create(context.TODO(), ingress, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func generateAPIServerIngress(name, namespace string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: pointer.String(IngressClassNameNGINX),
			Rules: []networkingv1.IngressRule{
				{
					Host: util.GenerateDevLocalDNSName(name),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: &pathTypePrefix,
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
											Port: networkingv1.ServiceBackendPort{
												Number: SecurePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
