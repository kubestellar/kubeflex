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
	// "fmt"

	_ "embed"
	// "github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"

	// "github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/client-go/tools/clientcmd"
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

// KubeconfigSecret containing k3s kubeconfig and incluster kubeconfig
type KubeconfigSecret struct {
	*shared.BaseReconciler
	Object *v1.Secret
}

// NewKubeconfigSecret for k3s server
func NewKubeconfigSecret(br *shared.BaseReconciler) *KubeconfigSecret {
	return &KubeconfigSecret{
		BaseReconciler: br,
		Object:         &v1.Secret{},
	}
}

// Prepare kubeconfig secret object and its manifest
func (r *KubeconfigSecret) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	r.Object = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      KubeconfigSecretName,
			Namespace: ComputeSystemNamespaceName(hcp.Name),
		},
		Data: map[string][]byte{
			KubeconfigSecretKey:          {},
			KubeconfigSecretKeyInCluster: {},
		},
	}
	return nil
}

// Reconcile a secret
// implements ControlPlaneReconciler
func (r *KubeconfigSecret) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("kubeconfig secret is already created", "secret", r.Object.Name)
	case apierrors.IsNotFound(err):
		// Create new secret
		if err := controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "setting k3s secret controller reference failed")
			return ctrl.Result{}, err
		}
		if err = r.Client.Create(context.TODO(), r.Object); err != nil {
			log.Error(err, "failed to create k3s secret")
			return ctrl.Result{}, err
		}
		log.Info("k3s secret is successfully created", "secretName", r.Object.Name)
	default:
		log.Error(err, "reconcile kubeconfig secret has failed")
		return ctrl.Result{}, err
	}
	log.Info("reconcile secret data to inject in client kubeconfig")
	// Store hosting cluster kubeconfig
	// kconf, err := clientcmd.Load(r.Object.Data[KubeconfigSecretKey])
	// if err != nil {
	// 	log.Error(err, "failed to load kubeconfig from secret", "secretName", KubeconfigSecretName, "secretKey", KubeconfigSecretKey)
	// 	return ctrl.Result{}, err
	// }
	// if string(r.Object.Data[KubeconfigSecretKey]) == "" {
	// 	err = fmt.Errorf("kubeconfig secret is empty")
	// 	log.Error(err, "secret data is empty", "secretName", r.Object.Name, "secretKey", KubeconfigSecretKey)
	// 	return ctrl.Result{RequeueAfter: RetryAfterDuration}, err
	// }
	// // Change cluster.server value of loopback with static DNS record
	// for cluster := range kconf.Clusters {
	// 	kconf.Clusters[cluster].Server = GetInClusterStaticDNSRecord(namespace)
	// }
	// // Store k3s incluster kubeconfig
	// inClusterConfigYAML, err := clientcmd.Write(*kconf)
	// if err != nil {
	// 	log.Error(err, "failed to write k3s kubeconfig")
	// 	return ctrl.Result{}, err
	// }
	// r.Object.Data[KubeconfigSecretKeyInCluster] = inClusterConfigYAML
	// log.Info("k3s in-cluster kubeconfig has new values, but not updated yet")
	// if err = r.Client.Update(context.TODO(), r.Object); err != nil {
	// 	log.Error(err, "on failure during update attempt of secret", "secretName", r.Object.Name)
	// 	return ctrl.Result{}, err
	// }
	// log.Info("k3s secret is successfully updated", "secretName", r.Object.Name)
	return ctrl.Result{}, nil
}

const (
	ScriptsConfigMapName               = "k3s-scripts"
	ScriptSaveKubeconfigIntoSecretName = "save-k3s-kubeconfig.sh" // must be the same as go:embed
	ScriptSaveTokenIntoSecretName      = "save-k3s-token.sh"      // must be the same as go:embed
	ScriptSaveCertsIntoSecretName      = "save-k3s-certs.sh"      // must be the same as go:embed
)

//go:embed embed/save-k3s-kubeconfig.sh
var saveKubeconfigIntoSecretScript string

//go:embed embed/save-k3s-token.sh
var saveTokenIntoSecretScript string

// ScriptsConfigMap containing data for k3s
type ScriptsConfigMap struct {
	*shared.BaseReconciler
	Object *v1.ConfigMap
}

// NewScriptsConfigMap create a config map with k3s utility scripts
func NewScriptsConfigMap(br *shared.BaseReconciler) *ScriptsConfigMap {
	return &ScriptsConfigMap{
		BaseReconciler: br,
		Object:         &v1.ConfigMap{},
	}
}

// Prepare kubeconfig secret object and its manifest
func (r *ScriptsConfigMap) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	r.Object = &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ScriptsConfigMapName,
			Namespace: ComputeSystemNamespaceName(hcp.Name),
		},
		Data: map[string]string{
			ScriptSaveKubeconfigIntoSecretName: saveKubeconfigIntoSecretScript, // add bash script content
			ScriptSaveTokenIntoSecretName:      saveTokenIntoSecretScript,      // add bash script content
		},
	}
	return nil
}

// Reconcile configmap
// implements ControlPlaneReconciler
func (r *ScriptsConfigMap) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("Reconcile k3s configmap")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("configmap is already created", "configmap", r.Object.Name)
	case apierrors.IsNotFound(err):
		// create a new config map
		log.Info("configmap is not found, creating new configmap")
		// Set Controller Reference on configmap
		if err := controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "setting k3s scripts configmap controller reference failed")
			return ctrl.Result{}, err
		}
		if err = r.Client.Create(context.TODO(), r.Object); err != nil {
			log.Error(err, "failed to create configmap", "configmap", r.Object.Name)
			return ctrl.Result{}, err
		}
		log.Info("configmap is created", "configmap", r.Object.Name)
	default:
		log.Error(err, "k3s configmap reconcile has failed")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
