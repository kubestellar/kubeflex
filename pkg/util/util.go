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
	"strings"

	"github.com/kubestellar/kubeflex/pkg/client"
)

const (
	ProjectName       = "kubeflex"
	DBReleaseName     = "postgres"
	SystemNamespace   = "kubeflex-system"
	IngressSecurePort = "9443"
)

func GenerateNamespaceFromControlPlaneName(name string) string {
	return fmt.Sprintf("%s-system", name)
}

// GenerateDevLocalDNSName: generates the local dns name for test/dev
// from the controlplane name
func GenerateDevLocalDNSName(name string) string {
	// At this time we use localtest.me for resolving to localhost.
	// TODO: make this configurable so that user can pick his preferred provider.
	return fmt.Sprintf("%s.localtest.me", name)
}

func GeneratePSecretName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}

func GeneratePSReplicaSetName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}

func GenerateOperatorDeploymentName() string {
	return fmt.Sprintf("%s-controller-manager", ProjectName)
}

func ParseVersionNumber(versionString string) string {
	versionParts := strings.Split(versionString, ".")
	return versionParts[0] + "." + versionParts[1] + "." + versionParts[2]
}

func GetKubernetesClusterVersionInfo(kubeconfig string) (string, error) {
	clientSet := client.GetClientSet(kubeconfig)
	serverVersion, err := clientSet.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return serverVersion.String(), nil
}
