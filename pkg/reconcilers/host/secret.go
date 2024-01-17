/*
Copyright 2024 The KubeStellar Authors.

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

package host

import (
	"context"
	"fmt"

	"github.com/kubestellar/kubeflex/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

const (
	DEFAULT_KUBE_APISERVER_ENDPOINT = "https://kubernetes.default.svc"
)

func (r *HostReconciler) ReconcileServiceAccountSecret(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// create service account secret
	saSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hcp.Name,
			Namespace:   namespace,
			Annotations: map[string]string{"kubernetes.io/service-account.name": hcp.Name},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(saSecret), saSecret, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := controllerutil.SetControllerReference(hcp, saSecret, r.Scheme); err != nil {
				return nil
			}
			err = r.Client.Create(context.TODO(), saSecret, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func (r *HostReconciler) ReconcileKubeconfigSecret(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	ca, token, err := r.getServiceAccountToken(ctx, hcp)
	if err != nil {
		return err
	}

	conf := generateConfig(ca, token, hcp.Name)
	confBytes, err := clientcmd.Write(*conf)
	if err != nil {
		return err
	}

	// create kubeconfig secret
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AdminConfSecret,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{util.KubeconfigSecretKeyInCluster: confBytes},
	}

	err = r.Client.Get(context.TODO(), client.ObjectKeyFromObject(kubeConfigSecret), kubeConfigSecret, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := controllerutil.SetControllerReference(hcp, kubeConfigSecret, r.Scheme); err != nil {
				return nil
			}
			err = r.Client.Create(context.TODO(), kubeConfigSecret, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

// retrieve the CA and the token from the service account secret
func (r *HostReconciler) getServiceAccountToken(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) ([]byte, []byte, error) {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// get service account secret
	saSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Name,
			Namespace: namespace,
		},
	}

	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(saSecret), saSecret, &client.GetOptions{}); err != nil {
		return nil, nil, err
	}

	// if the token is not present need to throw an error so it will retry
	saTokenBytes, ok := saSecret.Data["token"]
	if !ok {
		return nil, nil, fmt.Errorf("token not ready yet for service account secret %s, requeing", hcp.Name)
	}

	// if the ca is not present need to throw an error so it will retry
	saCertifcateAuthorityBytes, ok := saSecret.Data["ca.crt"]
	if !ok {
		return nil, nil, fmt.Errorf("ca not ready yet for service account secret %s, requeing", hcp.Name)
	}

	return saCertifcateAuthorityBytes, saTokenBytes, nil
}

func generateConfig(ca, token []byte, name string) *clientcmdapi.Config {
	config := clientcmdapi.NewConfig()
	config.Clusters[name] = &clientcmdapi.Cluster{
		Server:                   DEFAULT_KUBE_APISERVER_ENDPOINT,
		CertificateAuthorityData: ca,
	}
	config.AuthInfos[name] = &clientcmdapi.AuthInfo{
		Token: string(token),
	}
	config.Contexts[name] = &clientcmdapi.Context{
		Cluster:  name,
		AuthInfo: name,
	}
	config.CurrentContext = name
	return config
}
