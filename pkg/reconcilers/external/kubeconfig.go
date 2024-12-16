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

package external

import (
	"context"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	adoptedClusterSAName      = "kubeflex"
	adoptedClusterSANamespace = "kube-system"
)

var (
	defaultAdoptedTokenExpirationSeconds int64 = 365 * 86400
)

func (r *ExternalReconciler) ReconcileKubeconfigFromBoostrapSecret(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)

	// do not reconcile if kubeconfig secret is already present
	if r.IsKubeconfigSecretPresent(ctx, *hcp) {
		return nil
	}

	bootstrapApiConfig, err := getKubeconfigFromBoostrapSecret(r.Client, ctx, hcp)
	if err != nil {
		return err
	}

	bootstrapRestConfig, err := clientcmd.NewDefaultClientConfig(*bootstrapApiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return err
	}

	aClientset, err := kubernetes.NewForConfig(bootstrapRestConfig)
	if err != nil {
		return err
	}

	if err = reconcileAdoptedClusterRoleBinding(aClientset, adoptedClusterSAName, adoptedClusterSANamespace); err != nil {
		return fmt.Errorf("error creating ClusterRoleBinding on the adopted cluster: %v", err)
	}

	if err = reconcileAdoptedServiceAccount(aClientset.CoreV1().ServiceAccounts(adoptedClusterSANamespace), adoptedClusterSAName); err != nil {
		return fmt.Errorf("error creating ServiceAccount on the adopted cluster: %v", err)
	}

	bearerToken, err := requestTokenWithExpiration(aClientset.CoreV1().ServiceAccounts(adoptedClusterSANamespace), adoptedClusterSAName, hcp.Spec.AdoptedTokenExpirationSeconds)
	if err != nil {
		return fmt.Errorf("error requesting token from the adopted cluster: %v", err)
	}

	newKubeConfig, err := createNewKubeConfig(bootstrapApiConfig, bearerToken, hcp)
	if err != nil {
		return fmt.Errorf("error creating adopted cluster kubeconfig: %v", err)
	}

	if err := r.ReconcileKubeconfigSecret(ctx, *hcp, newKubeConfig); err != nil {
		return fmt.Errorf("error creating kubeconfig secret: %v", err)
	}

	return deleteBoostrapSecret(r.Client, ctx, hcp)
}

func getKubeconfigFromBoostrapSecret(crClient client.Client, ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (*api.Config, error) {
	_ = clog.FromContext(ctx)

	if hcp.Spec.BootstrapSecretRef == nil {
		return nil, fmt.Errorf("bootstrapSecretRef must be present in the control plane")
	}

	bootstrapSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Spec.BootstrapSecretRef.Name,
			Namespace: hcp.Spec.BootstrapSecretRef.Namespace,
		},
	}

	err := crClient.Get(context.TODO(), client.ObjectKeyFromObject(bootstrapSecret), bootstrapSecret, &client.GetOptions{})
	if err != nil {
		return nil, err
	}

	key := util.DefaultString(hcp.Spec.BootstrapSecretRef.Key, util.KubeconfigSecretKeyInCluster)

	kconfigBytes := bootstrapSecret.Data[key]
	if kconfigBytes == nil {
		return nil, fmt.Errorf("kubeconfig not found in bootstrap secret for key %s", key)
	}

	return clientcmd.Load(kconfigBytes)
}

func reconcileAdoptedClusterRoleBinding(clientset *kubernetes.Clientset, saName, saNamespace string) error {
	name := saName + "-clusterrolebinding"

	// Check if the ClusterRoleBinding already exists
	_, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	// Define new ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: saNamespace,
			},
		},
	}

	// Create the ClusterRoleBinding
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func reconcileAdoptedServiceAccount(saClient v1.ServiceAccountInterface, saName string) error {

	// Check if the ServiceAccount already exists
	_, err := saClient.Get(context.TODO(), saName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	// Define a new ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
		},
	}

	// Create the ServiceAccount
	_, err = saClient.Create(context.TODO(), sa, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func requestTokenWithExpiration(saClient v1.ServiceAccountInterface, saName string, expirationSeconds *int64) (string, error) {
	if expirationSeconds == nil {
		expirationSeconds = &defaultAdoptedTokenExpirationSeconds
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{},
			ExpirationSeconds: expirationSeconds,
		},
	}

	tokenResponse, err := saClient.CreateToken(context.TODO(), saName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return tokenResponse.Status.Token, nil
}

func createNewKubeConfig(bootstrapConfig *api.Config, token string, hcp *tenancyv1alpha1.ControlPlane) ([]byte, error) {
	context := bootstrapConfig.Contexts[bootstrapConfig.CurrentContext]

	cluster := bootstrapConfig.Clusters[context.Cluster]
	if cluster == nil {
		return nil, fmt.Errorf("invalid cluster name %s for adoped cluster", context.Cluster)
	}

	kubeConfig := api.NewConfig()

	kubeConfig.Clusters[context.Cluster] = &api.Cluster{
		Server:                   cluster.Server,
		CertificateAuthorityData: cluster.CertificateAuthorityData,
	}

	kubeConfig.AuthInfos[hcp.Name] = &api.AuthInfo{
		Token: token,
	}

	kubeConfig.Contexts[hcp.Name] = &api.Context{
		Cluster:  context.Cluster,
		AuthInfo: hcp.Name,
	}
	kubeConfig.CurrentContext = hcp.Name

	newKubeConfig, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return nil, err
	}

	return newKubeConfig, nil
}

func (r *ExternalReconciler) ReconcileKubeconfigSecret(ctx context.Context, cp tenancyv1alpha1.ControlPlane, kubeconfig []byte) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(cp.Name)

	// create kubeconfig secret
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AdminConfSecret,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{util.KubeconfigSecretKeyInCluster: kubeconfig},
	}

	// Attempt to get the existing kubeconfig secret
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(kubeConfigSecret), kubeConfigSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Set controller reference on new kubeconfig secret
			if setErr := controllerutil.SetControllerReference(&cp, kubeConfigSecret, r.Scheme); setErr != nil {
				return setErr
			}
			// Create the kubeconfig secret as it does not exist
			return r.Client.Create(ctx, kubeConfigSecret)
		}
		return err
	}

	// Update the existing kubeconfig secret
	return r.Client.Update(ctx, kubeConfigSecret)
}

func deleteBoostrapSecret(crClient client.Client, ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)

	bootstrapSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcp.Spec.BootstrapSecretRef.Name,
			Namespace: hcp.Spec.BootstrapSecretRef.Namespace,
		},
	}
	return crClient.Delete(context.TODO(), bootstrapSecret)
}

func (r *ExternalReconciler) IsKubeconfigSecretPresent(ctx context.Context, cp tenancyv1alpha1.ControlPlane) bool {
	namespace := util.GenerateNamespaceFromControlPlaneName(cp.Name)

	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.AdminConfSecret,
			Namespace: namespace,
		},
	}

	// Attempt to get the existing kubeconfig secret
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(kubeConfigSecret), kubeConfigSecret); err == nil {
		return true
	}

	return false
}
