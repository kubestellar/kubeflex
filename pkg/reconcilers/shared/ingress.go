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

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	// IngressClassNameNGINX is the name of the NGINX ingress class used for routing traffic.
	IngressClassNameNGINX = "nginx"
)

var (
	// pathTypePrefix specifies that the path type should match based on the provided prefix.
	pathTypePrefix = networkingv1.PathTypePrefix
)

// ReconcileAPIServerIngress ensures that an Ingress resource exists for the API server of the given ControlPlane.
// If the Ingress does not exist, it is created. If a transient error occurs, it will be retried.
//
// Parameters:
//   - ctx: Context for request-scoped values and deadlines.
//   - hcp: The target ControlPlane resource for which the Ingress should be reconciled.
//   - svcName: The name of the Kubernetes Service backing the API server (defaults to hcp.Name if empty).
//   - svcPort: The port number on which the Service is exposed.
//   - domain: The domain used to generate the Ingress host.
func (r *BaseReconciler) ReconcileAPIServerIngress(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, svcName string, svcPort int, domain string) error {
	// Namespace is derived from the ControlPlane's name.
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Default service name to ControlPlane name if not provided.
	if svcName == "" {
		svcName = hcp.Name
	}

	// Attempt to fetch the existing Ingress resource.
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(ingress), ingress, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create a new Ingress if it doesn't exist.
			ingress = generateAPIServerIngress(hcp.Name, svcName, namespace, svcPort, domain)
			if err := controllerutil.SetControllerReference(hcp, ingress, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			if err = r.Client.Create(ctx, ingress, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return err // Retry transient errors
				}
				return fmt.Errorf("failed to create ingress: %w", err)
			}
		} else if util.IsTransientError(err) {
			return err // Retry transient errors
		} else {
			return fmt.Errorf("failed to get ingress: %w", err)
		}
	}
	return nil
}

// generateAPIServerIngress creates a new Ingress resource for the API server.
// The Ingress routes traffic to the specified service and port, using the provided domain name.
//
// Parameters:
//   - name: Name of the Ingress resource.
//   - svcName: Name of the Service that backs the API server.
//   - namespace: Namespace in which the Ingress will be created.
//   - svcPort: Port number of the Service.
//   - domain: Domain used to generate the Ingress hostname.
func generateAPIServerIngress(name, svcName, namespace string, svcPort int, domain string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				// Enable SSL passthrough for NGINX ingress to allow TLS termination at the API server.
				"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: pointer.String(IngressClassNameNGINX),
			Rules: []networkingv1.IngressRule{
				{
					Host: util.GenerateDevLocalDNSName(name, domain),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									PathType: &pathTypePrefix,
									Path:     "/",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: svcName,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(svcPort),
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
