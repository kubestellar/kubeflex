/*
Package k3s

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
	Object *networkingv1.Ingress
}

// NewIngress create k3s ingress to reach k3s apiserver from outside the cluster
func NewIngress(br *shared.BaseReconciler) *Ingress {
	return &Ingress{
		BaseReconciler: br,
		Object:         &networkingv1.Ingress{},
	}
}

// Prepare ingress and its manifest
func (r *Ingress) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	// NOTE: host cannot have https:// prefix - see RFC 1123
	log := clog.FromContext(ctx)
	cfg, err := r.BaseReconciler.GetConfig(ctx)
	if err != nil {
		log.Error(err, "missing shared configuration kubeflex configmap")
		return err
	}

	r.Object = &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: ComputeSystemNamespaceName(hcp.Name),
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr.To(IngressClassNameNGINX),
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", hcp.Name, cfg.Domain),

					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: ptr.To(networkingv1.PathTypePrefix),
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: ServiceName,
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
	return nil
}

// Reconcile the ingress
func (r *Ingress) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("reconciling k3s ingress")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("k3s ingress is already created", "ingress", r.Object.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s ingress failed to be fetched")
		if err = controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference on ingress")
			return ctrl.Result{}, err
		}
		// Create new ingress on the cluster
		if err = r.Client.Create(ctx, r.Object); err != nil {
			log.Error(err, "failed to create k3s ingress on the cluster")
		}
		log.Info("k3s ingress is successfully created")
	default:
		log.Error(err, "k3s ingress reconcile has failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
