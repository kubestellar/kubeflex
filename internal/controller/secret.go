package controller

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"mcc.ibm.org/kubeflex/pkg/certs"
	"mcc.ibm.org/kubeflex/pkg/util"
)

func (r *ControlPlaneReconciler) ReconcileCertsSecret(ctx context.Context, name string, owner *metav1.OwnerReference) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	// create certs secret object
	csecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certs.CertsSecretName,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(csecret), csecret, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			csecret, err := generateCertsSecret(ctx, namespace)
			if err != nil {
				return err
			}
			util.EnsureOwnerRef(csecret, owner)
			err = r.Client.Create(context.TODO(), csecret, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func generateCertsSecret(ctx context.Context, namespace string) (*v1.Secret, error) {
	c, err := certs.New(ctx)
	if err != nil {
		return nil, err
	}
	return c.GenerateCertsSecret(ctx, namespace), nil
}
