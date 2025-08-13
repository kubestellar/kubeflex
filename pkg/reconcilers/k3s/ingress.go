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
package k3s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	IngressClassNameNGINX = "nginx"
)

type Ingress struct {
	*shared.BaseReconciler
}

// NewIngress create k3s ingress to reach k3s apiserver from outside the cluster
func NewIngress(host string, serviceName string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServerName,
			Namespace: ServerSystemNamespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To(IngressClassNameNGINX),
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: ptr.To(networkingv1.PathTypePrefix),
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: shared.DefaultPort,
												// Name:   shared.DefaultPortName,
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

// Reconcile the ingress
func (ingress *Ingress) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	log.Info("reconciling k3s ingress")
	// Get config to init Ingress
	cfg, err := ingress.BaseReconciler.GetConfig(ctx)
	if err != nil {
		log.Error(err, "missing shared configuration kubeflex configmap")
		return ctrl.Result{}, err
	}
	// NOTE: host cannot have https:// prefix - see RFC 1123
	ingrHost := fmt.Sprintf("%s.%s", ServiceName, cfg.Domain)
	ingr := NewIngress(ingrHost, hcp.Name)
	// Get ingress on cluster to verify its existence
	err = ingress.Client.Get(ctx, client.ObjectKeyFromObject(ingr), ingr)
	switch {
	case err == nil:
		log.Info("k3s ingress is already created", "ingress", ingrHost)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s ingress failed to be fetched")
		if err = controllerutil.SetControllerReference(hcp, ingr, ingress.Scheme); err != nil {
			log.Error(err, "k3s ingress failed to set controller reference")
			return ctrl.Result{}, err
		}
		// Create new ingress on the cluster
		if err = ingress.Client.Create(ctx, ingr); err != nil {
			log.Error(err, "ingress.go: failed to create k3s ingress on the cluster")
		}
		log.Info("k3s ingress is successfully created")
	default:
		log.Error(err, "k3s ingress reconcile has failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
