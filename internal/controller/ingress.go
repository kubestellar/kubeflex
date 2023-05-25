package controller

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"mcc.ibm.org/kubeflex/pkg/util"
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
