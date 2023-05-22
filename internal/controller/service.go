package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"mcc.ibm.org/kubeflex/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ControlPlaneReconciler) ReconcileAPIServerService(ctx context.Context, name string, owner *metav1.OwnerReference) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	// create service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(service), service, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			service := generateAPIServerService(name, namespace)
			util.EnsureOwnerRef(service, owner)
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
			// Type: "NodePort",
			Selector: map[string]string{
				"app": APIServerDeploymentName,
			},
			Ports: []corev1.ServicePort{
				// {
				// 	Port: 80,
				// 	//NodePort: 30001,
				// 	Name:     "http",
				// 	Protocol: "TCP",
				// },
				{
					Port: SecurePort,
					//NodePort: 30002,
					Name:     "https",
					Protocol: "TCP",
				},
			},
		},
	}
}
