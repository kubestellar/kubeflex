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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	GatewayClassName = "nginx-gateway-fabric"
)

type Gateway struct {
	*shared.BaseReconciler
	Object *gatewayv1.HTTPRoute
}

// NewGateway create k3s HTTPRoute to reach k3s apiserver from outside the cluster
func NewGateway(br *shared.BaseReconciler) *Gateway {
	return &Gateway{
		BaseReconciler: br,
		Object:         &gatewayv1.HTTPRoute{},
	}
}

// Prepare HTTPRoute and its manifest
func (r *Gateway) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	log := clog.FromContext(ctx)
	cfg, err := r.BaseReconciler.GetConfig(ctx)
	if err != nil {
		log.Error(err, "missing shared configuration kubeflex configmap")
		return err
	}

	hostname := gatewayv1.Hostname(fmt.Sprintf("%s.%s", hcp.Name, cfg.Domain))

	r.Object = &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: "gateway.networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: ComputeSystemNamespaceName(hcp.Name),
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      "kubeflex-gateway",
						Namespace: (*gatewayv1.Namespace)(ptr.To("nginx-gateway")),
					},
				},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  ptr.To(gatewayv1.PathMatchPathPrefix),
								Value: ptr.To("/"),
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(ServiceName),
									Port: (*gatewayv1.PortNumber)(ptr.To(gatewayv1.PortNumber(shared.DefaultPort))),
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

// Reconcile the HTTPRoute
func (r *Gateway) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("reconciling k3s httproute")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("k3s httproute is already created", "httproute", r.Object.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s httproute failed to be fetched")
		if err = controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "failed to set controller reference on httproute")
			return ctrl.Result{}, err
		}
		// Create new HTTPRoute on the cluster
		if err = r.Client.Create(ctx, r.Object); err != nil {
			log.Error(err, "failed to create k3s httproute on the cluster")
		}
		log.Info("k3s httproute is successfully created")
	default:
		log.Error(err, "k3s httproute reconcile has failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
