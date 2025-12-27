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

package shared

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (r *BaseReconciler) ReconcileGatewayClass(ctx context.Context) error {
	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: GatewayClassName,
		},
	}

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(gatewayClass), gatewayClass, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			gatewayClass = generateGatewayClass()
			if err = r.Client.Create(ctx, gatewayClass, &client.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create gatewayclass: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get gatewayclass: %w", err)
		}
	}
	return nil
}

func (r *BaseReconciler) ReconcileGateway(ctx context.Context) error {
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayName,
			Namespace: "nginx-gateway",
		},
	}

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(gateway), gateway, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			gateway = generateGateway()
			if err = r.Client.Create(ctx, gateway, &client.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create gateway: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get gateway: %w", err)
		}
	}
	return nil
}

func generateGatewayClass() *gatewayv1.GatewayClass {
	return &gatewayv1.GatewayClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "GatewayClass",
			APIVersion: "gateway.networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: GatewayClassName,
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.nginx.org/nginx-gateway-controller",
		},
	}
}

func generateGateway() *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: "gateway.networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      GatewayName,
			Namespace: "nginx-gateway",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: GatewayClassName,
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr.To(gatewayv1.NamespacesFromAll),
						},
					},
				},
				{
					Name:     "https",
					Protocol: gatewayv1.HTTPSProtocolType,
					Port:     443,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr.To(gatewayv1.NamespacesFromAll),
						},
					},
				},
			},
		},
	}
}
