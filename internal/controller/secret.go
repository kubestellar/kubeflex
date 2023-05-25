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

func (r *ControlPlaneReconciler) ReconcileCertsSecret(ctx context.Context, name string, owner *metav1.OwnerReference) (*certs.Certs, error) {
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
			csecret, crts, err := generateCertsSecret(ctx, name, namespace)
			if err != nil {
				return nil, err
			}
			util.EnsureOwnerRef(csecret, owner)
			err = r.Client.Create(context.TODO(), csecret, &client.CreateOptions{})
			if err != nil {
				return nil, err
			}
			return crts, nil
		}
		return nil, err
	}
	return nil, nil
}

func (r *ControlPlaneReconciler) ReconcileKubeconfigSecret(ctx context.Context, crts *certs.Certs, conf certs.ConfigGen, owner *metav1.OwnerReference) error {
	// TODO - temp hack - we should make this independent of the certs gen.
	// Should gen kconfig from certs secret otherwise it may fail if certs are not generated before this func
	if crts == nil {
		return nil
	}
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(conf.CpName)

	// create certs secret object
	conf.CpNamespace = namespace
	csecret, err := certs.GenerateKubeConfigSecret(ctx, crts, conf)
	if err != nil {
		return err
	}

	err = r.Client.Get(context.TODO(), client.ObjectKeyFromObject(csecret), csecret, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
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

func generateCertsSecret(ctx context.Context, name, namespace string) (*v1.Secret, *certs.Certs, error) {
	c, err := certs.New(ctx, []string{name, util.GenerateDevLocalDNSName(name)})
	if err != nil {
		return nil, nil, err
	}
	return c.GenerateCertsSecret(ctx, namespace), c, nil
}
