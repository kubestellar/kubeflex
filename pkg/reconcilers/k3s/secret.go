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

package k3s

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/k3s"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const KubeconfigSecretName = "k3s-config"
const KubeconfigSecretKey = "config"
const KubeconfigSecretKeyInCluster = "config-incluster"

// K3s service
type Secret struct {
	*shared.BaseReconciler
}

// Init secret for k3s server
func NewKubeconfigSecret(namespace string) (_ *v1.Secret, err error) {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeconfigSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			KubeconfigSecretKey:          {},
			KubeconfigSecretKeyInCluster: {},
		},
	}, nil
}

func handleReconcileError(log logr.Logger, err error) (ctrl.Result, error) {
	if util.IsTransientError(err) {
		// Retry
		log.Error(err, "secret reconcile is on transient err, retrying now")
		return ctrl.Result{Requeue: true}, err // Retry transient errors
	}
	if err != nil {
		log.Error(err, "secret reconcile is on err", "error", err)
		return ctrl.Result{}, fmt.Errorf("failed to reconcile: %w", err)
	}
	return ctrl.Result{}, nil
}

// Reconcile a secret
// implements ControlPlaneReconciler
func (r *Secret) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	namespace := GenerateSystemNamespaceName(hcp.Name)
	ksecret, _ := NewKubeconfigSecret(namespace)
	// Get secret from cluster if existent
	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(ksecret), ksecret, &client.GetOptions{})
	if secretIsNotFound := apierrors.IsNotFound(err); !secretIsNotFound {
		return handleReconcileError(log, err)
	}
	log.Info("secret is found on the kubernetes cluster", "secretName", ksecret.Name)
	// Store hosting cluster kubeconfig
	kconf, err := clientcmd.Load(ksecret.Data[KubeconfigSecretKey])
	if err != nil {
		log.Error(err, "failed to load kubeconfig from secret", "secretName", KubeconfigSecretName, "secretKey", KubeconfigSecretKey)
		return ctrl.Result{}, err
	}
	for cluster := range kconf.Clusters {
		// Update cluster by adding
		kconf.Clusters[cluster].Server = k3s.GetStaticDNSRecord(namespace)
	}
	// Store k3s incluster kubeconfig
	if inClusterConfigYAML, err := clientcmd.Write(*kconf); err == nil {
		ksecret.Data[KubeconfigSecretKeyInCluster] = inClusterConfigYAML
		log.Info("hosting cluster kubeconfig successfully saved")
		log.Info("k3s in-cluster kubeconfig successfully saved")
	} else {
		log.Error(err, "failed to write k3s kubeconfig")
		return ctrl.Result{}, err
	}
	// Create new secret
	err = r.Client.Update(context.TODO(), ksecret)
	if res, err := handleReconcileError(log, err); err != nil {
		return res, err
	}
	log.Info("secret is successfully created", "secretName", ksecret.Name)
	return r.BaseReconciler.Reconcile(ctx, hcp)
}
