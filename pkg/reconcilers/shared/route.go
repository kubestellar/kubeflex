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
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func (r *BaseReconciler) ReconcileAPIServerRoute(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, svcName string, svcPort int, domain string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	if svcName == "" {
		svcName = hcp.Name
	}

	// lookup route
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(route), route, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			route = generateAPIServerRoute(hcp.Name, svcName, namespace, svcPort, domain)
			if err := controllerutil.SetControllerReference(hcp, route, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			if err = r.Client.Create(ctx, route, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return err // Retry transient errors
				}
				return fmt.Errorf("failed to create route: %w", err)
			}
		} else if util.IsTransientError(err) {
			return err // Retry transient errors
		} else {
			return fmt.Errorf("failed to get route: %w", err)
		}
	}
	return nil
}

func generateAPIServerRoute(name, svcName, namespace string, svcPort int, domain string) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"openshift.io/host.generated": "true",
			},
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   svcName,
				Weight: pointer.Int32(100),
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(DefaultPortName),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationPassthrough,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
}

func (r *BaseReconciler) GetAPIServerRouteURL(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (string, error) {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// lookup route
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(route), route, &client.GetOptions{})
	if err != nil {
		return "", err
	}

	return route.Spec.Host, nil
}
