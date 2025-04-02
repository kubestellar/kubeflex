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

package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

func (r *K8sReconciler) ReconcileCertsSecret(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, cfg *shared.SharedConfig, extraDNSName string) (*certs.Certs, error) {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

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
			csecret, crts, err := generateCertsSecret(ctx, hcp.Name, namespace, cfg.Domain, extraDNSName)
			if err != nil {
				return nil, err
			}
			if err := controllerutil.SetControllerReference(hcp, csecret, r.Scheme); err != nil {
				return nil, err
			}
			if err = r.Client.Create(context.TODO(), csecret, &client.CreateOptions{}); err != nil {
				return nil, err
			}
			return crts, nil
		}
		return nil, err
	}
	return nil, nil
}

func (r *K8sReconciler) ReconcileKubeconfigSecret(ctx context.Context, crts *certs.Certs, conf *certs.ConfigGen, hcp *tenancyv1alpha1.ControlPlane) error {
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
			if err := controllerutil.SetControllerReference(hcp, csecret, r.Scheme); err != nil {
				return err
			}
			if err = r.Client.Create(context.TODO(), csecret, &client.CreateOptions{}); err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func generateCertsSecret(ctx context.Context, name, namespace, domain, extraDNSName string) (*v1.Secret, *certs.Certs, error) {
	extraDnsNames := util.GenerateHostedDNSName(namespace, name)
	extraDnsNames = append(extraDnsNames, util.GenerateDevLocalDNSName(name, domain))
	if extraDNSName != "" {
		extraDnsNames = append(extraDnsNames, extraDNSName)
	}
	c, err := certs.New(ctx, extraDnsNames)
	if err != nil {
		return nil, nil, err
	}
	return c.GenerateCertsSecret(ctx, namespace), c, nil
}
