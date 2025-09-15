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

package util

import (
	"fmt"
	"os"
	"strings"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	APIServerDeploymentName              = "kube-apiserver"            // TODO: move to its reconciler component (k8s)
	OCMServerDeploymentName              = "multicluster-controlplane" // TODO: move to its reconciler component (ocm)
	VClusterServerDeploymentName         = "vcluster"                  // TODO: move to its reconciler component (vcluster)
	CMDeploymentName                     = "kube-controller-manager"
	ProjectName                          = "kubeflex"
	DBReleaseName                        = "postgres"
	SystemNamespace                      = "kubeflex-system"
	SystemConfigMap                      = "kubeflex-config"
	AdminConfSecret                      = "admin-kubeconfig"                     // TODO to replace as it is defined in reconciler component (k3s)
	OCMKubeConfigSecret                  = "multicluster-controlplane-kubeconfig" // TODO to replace as it is defined in reconciler component (k3s)
	VClusterKubeConfigSecret             = "vc-vcluster"                          // TODO to replace as it is defined in reconciler component (k3s)
	VClusterNodePortServiceName          = "vcluster-nodeport"                    // TODO to replace as it is defined in reconciler component (k3s)
	VClusterServiceName                  = "vcluster"                             // TODO to replace as it is defined in reconciler component (k3s)
	K3sKubeConfigSecret                  = "k3s-config"                           // TODO to replace as it is defined in reconciler component (k3s)
	K3sServerDeploymentName              = "k3s-server"                           // TODO to replace as it is defined in reconciler component (k3s)
	KubeconfigSecretKeyDefault           = "kubeconfig"
	KubeconfigSecretKeyInCluster         = "kubeconfig-incluster"
	KubeconfigSecretKeyVCluster          = "config"           // NOTE reuse by k3s
	KubeconfigSecretKeyVClusterInCluster = "config-incluster" // NOTE reuse by k3s
	NamespaceSuffix                      = "-system"          // TODO: change to SystemNamespaceSuffix
)

func GenerateNamespaceFromControlPlaneName(name string) string {
	return name + NamespaceSuffix
}

func ControlPlaneNameFromNamespace(nsName string) (string, error) {
	const suffixLen = len(NamespaceSuffix)
	nsLen := len(nsName)
	cpLen := nsLen - suffixLen
	if cpLen > 0 && nsName[cpLen:] == NamespaceSuffix {
		return nsName[:cpLen], nil
	}
	return "", fmt.Errorf("namespace %q does not end with %q", nsName, NamespaceSuffix)
}

// GenerateDevLocalDNSName: generates the local dns name for test/dev
// from the controlplane name
func GenerateDevLocalDNSName(name, domain string) string {
	return fmt.Sprintf("%s.%s", name, domain)
}

func GenerateHostedDNSName(namespace, name string) []string {
	return []string{
		fmt.Sprintf("%s.%s", name, namespace),
		fmt.Sprintf("%s.%s.svc", name, namespace),
		fmt.Sprintf("%s.%s.svc.cluster", name, namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace),
	}
}

func GenerateOperatorDeploymentName() string {
	return fmt.Sprintf("%s-controller-manager", ProjectName)
}

func ParseVersionNumber(versionString string) string {
	parts := strings.Split(versionString, ".")
	if len(parts) < 2 {
		fmt.Fprintf(os.Stderr, "WARNING: Unexpected version string format in ParseVersionNumber: %q\n", versionString)
		return versionString
	}
	return parts[1]
}

func GetKubernetesClusterVersionInfo(kubeconfig string) (string, error) {
	clientSet, err := client.GetClientSet(kubeconfig)
	if err != nil {
		return "", err
	}

	serverVersion, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return serverVersion.String(), nil
}

func GetKubeconfSecretNameByControlPlaneType(controlPlaneType string) string {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeK8S):
		return AdminConfSecret
	case string(tenancyv1alpha1.ControlPlaneTypeOCM):
		return OCMKubeConfigSecret
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		return VClusterKubeConfigSecret
	case string(tenancyv1alpha1.ControlPlaneTypeK3s):
		return K3sKubeConfigSecret
	default:
		// TODO - should we instead throw an error?
		return AdminConfSecret
	}
}

func GetKubeconfSecretKeyNameByControlPlaneType(controlPlaneType string) string {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeK8S), string(tenancyv1alpha1.ControlPlaneTypeOCM):
		return KubeconfigSecretKeyDefault
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster), string(tenancyv1alpha1.ControlPlaneTypeK3s):
		return KubeconfigSecretKeyVCluster
	default:
		// TODO - should we instead throw an error?
		return KubeconfigSecretKeyDefault
	}
}

func GetAPIServerDeploymentNameByControlPlaneType(controlPlaneType string) string {
	switch controlPlaneType {
	case string(tenancyv1alpha1.ControlPlaneTypeK8S):
		return APIServerDeploymentName
	case string(tenancyv1alpha1.ControlPlaneTypeOCM):
		return OCMServerDeploymentName
	case string(tenancyv1alpha1.ControlPlaneTypeVCluster):
		return VClusterServerDeploymentName
	case string(tenancyv1alpha1.ControlPlaneTypeK3s):
		return K3sServerDeploymentName
	default:
		// TODO - should we instead throw an error?
		return APIServerDeploymentName
	}
}

func IsInCluster() bool {
	if kubeHost := os.Getenv("KUBERNETES_SERVICE_HOST"); kubeHost != "" {
		return true
	}
	return false
}

func ZeroFields(obj runtime.Object) runtime.Object {
	zeroed := obj.DeepCopyObject()
	mObj := zeroed.(metav1.Object)
	mObj.SetManagedFields(nil)
	mObj.SetCreationTimestamp(metav1.Time{})
	mObj.SetGeneration(0)
	mObj.SetResourceVersion("")
	mObj.SetUID("")

	return zeroed
}

func DefaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func GenerateBootstrapSecretName(cpName string) string {
	return fmt.Sprintf("%s-bootstrap", cpName)
}
