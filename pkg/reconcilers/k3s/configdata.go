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

	_ "embed"
	"github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	KubeconfigSecretName         = "k3s-config"
	KubeconfigSecretKey          = "config"
	KubeconfigSecretKeyInCluster = "config-incluster"
)

// Secret containing k3s kubeconfig and incluster kubeconfig
type Secret struct {
	*shared.BaseReconciler
}

// NewKubeconfigSecret for k3s server
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
	if err != nil {
		if util.IsTransientError(err) {
			// Retry
			log.Error(err, "reconcile is on transient err, retrying now")
			return ctrl.Result{Requeue: true}, err // Retry transient errors
		} else {
			log.Error(err, "reconcile is on err", "error", err)
			return ctrl.Result{}, fmt.Errorf("failed to reconcile: %w", err)
		}
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
	if err != nil {
		// Secret does not exist
		if apierrors.IsNotFound(err) {
			// Create new secret
			if err := controllerutil.SetControllerReference(hcp, ksecret, r.Scheme); err != nil {
				log.Error(err, "setting k3s secret controller reference failed")
				return ctrl.Result{}, err
			}
			if err = r.Client.Create(context.TODO(), ksecret); err != nil {
				return handleReconcileError(log, err)
			}
			log.Info("k3s secret is successfully created", "secretName", ksecret.Name)
		} else {
			return handleReconcileError(log, err)
		}
	}
	// Secret exist
	log.Info("secret is found on the kubernetes cluster", "secretName", ksecret.Name)
	// Store hosting cluster kubeconfig
	kconf, err := clientcmd.Load(ksecret.Data[KubeconfigSecretKey])
	if err != nil {
		log.Error(err, "failed to load kubeconfig from secret", "secretName", KubeconfigSecretName, "secretKey", KubeconfigSecretKey)
		return ctrl.Result{}, err
	}
	// TODO: ksecret.Data[KubeconfigSecretKey] is empty, how to populate?
	if string(ksecret.Data[KubeconfigSecretKey]) == "" {
		log.Info("secret data is empty, populating its value...", "secretName", ksecret.Name, "secretKey", KubeconfigSecretKey)
	}
	for cluster := range kconf.Clusters {
		// Update cluster by adding
		kconf.Clusters[cluster].Server = GetInClusterStaticDNSRecord(namespace)
	}
	// Store k3s incluster kubeconfig
	inClusterConfigYAML, err := clientcmd.Write(*kconf)
	if err != nil {
		log.Error(err, "failed to write k3s kubeconfig")
		return ctrl.Result{}, err
	}
	ksecret.Data[KubeconfigSecretKeyInCluster] = inClusterConfigYAML
	log.Info("hosting cluster kubeconfig has new values, but not updated yet")
	log.Info("k3s in-cluster kubeconfig has new values, but not updated yet")
	if err = r.Client.Update(context.TODO(), ksecret); err != nil {
		log.Error(err, "on failure during update attempt of secret", "secretName", ksecret.Name)
		return handleReconcileError(log, err)
	}
	log.Info("k3s secret is successfully updated", "secretName", ksecret.Name)
	return r.BaseReconciler.Reconcile(ctx, hcp)
}

const (
	ScriptsConfigMapName               = "k3s-scripts"
	ScriptSaveKubeconfigIntoSecretName = "save-k3s-kubeconfig.sh"
	ScriptSaveCertsIntoSecretName      = "save-k3s-certs.sh"
)

//go:embed embed/save-k3s-kubeconfig.sh
var saveKubeconfigIntoSecretScript string

// ConfigMap containing data for k3s
type ConfigMap struct {
	*shared.BaseReconciler
}

// NewScriptsConfigMap create a config map with k3s utility scripts
func NewScriptsConfigMap(namespace string) (*v1.ConfigMap, error) {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ScriptsConfigMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			ScriptSaveKubeconfigIntoSecretName: saveKubeconfigIntoSecretScript, // add bash script content
		},
	}, nil
}

// Reconcile configmap
// implements ControlPlaneReconciler
func (cm *ConfigMap) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	log.Info("Reconcile k3s configmap")
	namespace := GenerateSystemNamespaceName(hcp.Name)
	cmScripts, _ := NewScriptsConfigMap(namespace)
	err := cm.Client.Get(ctx, client.ObjectKeyFromObject(cmScripts), cmScripts)
	switch {
	case err == nil:
		log.Info("configmap is already created", "configmap", cmScripts.Name)
	case apierrors.IsNotFound(err):
		// create a new config map
		log.Info("configmap is not found, creating new configmap")
		// Set Controller Reference on configmap
		if err := controllerutil.SetControllerReference(hcp, cmScripts, cm.Scheme); err != nil {
			log.Error(err, "setting k3s scripts configmap controller reference failed")
			return ctrl.Result{}, err
		}
		if err = cm.Client.Create(context.TODO(), cmScripts); err != nil {
			return handleReconcileError(log, err)
		}
		log.Info("configmap is created", "configmap", cmScripts.Name)
	default:
		log.Error(err, "k3s configmap reconcile has failed")
	}
	return ctrl.Result{}, nil
}
