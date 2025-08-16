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

package vcluster

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

// Reconcile the kubeconfig secret to set the `config-incluster` key with the in-cluster configuration
func (r *VClusterReconciler) ReconcileKubeconfigSecret(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	ksecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.VClusterKubeConfigSecret,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(ksecret), ksecret, &client.GetOptions{})
	if err != nil {
		if util.IsTransientError(err) {
			return err // Retry transient errors
		}
		return fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}

	kconfig := ksecret.Data[util.KubeconfigSecretKeyVCluster]
	if kconfig == nil {
		return fmt.Errorf("no kubeconfig found in vcluster kubeconfig secret %#v", ksecret)
	}

	config, err := clientcmd.Load(kconfig)
	if err != nil {
		return err
	}

	// there is only one cluster in the generated config, but to be on the safe
	// side update all
	for k := range config.Clusters {
		config.Clusters[k].Server = fmt.Sprintf("https://vcluster.%s", namespace)
	}

	inclusterConfig, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	// update secret and write it back
	ksecret.Data[util.KubeconfigSecretKeyVClusterInCluster] = inclusterConfig

	err = r.Client.Update(context.TODO(), ksecret, &client.UpdateOptions{})
	if err != nil {
		if util.IsTransientError(err) {
			return err // Retry transient errors
		}
		return fmt.Errorf("failed to update kubeconfig secret: %w", err)
	}

	return nil
}
