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

package ctx

import (
	"os"
	"testing"

	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigPath string = "./testconfig"
var hostingClusterContextMock = "kind-kubeflex"

// Setup mock kubeconfig file with context,cluster,authinfo
func setupMockContext(t *testing.T, kubeconfigPath string, ctxName string) {
	kconf := api.NewConfig()
	kconf.Contexts[hostingClusterContextMock] = &api.Context{
		Cluster:  hostingClusterContextMock,
		AuthInfo: hostingClusterContextMock,
	}
	kconf.Contexts[certs.GenerateContextName(ctxName)] = &api.Context{
		Cluster:  certs.GenerateClusterName(ctxName),
		AuthInfo: certs.GenerateAuthInfoAdminName(ctxName),
	}
	kconf.Clusters[certs.GenerateClusterName(ctxName)] = api.NewCluster()
	kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)] = api.NewAuthInfo()
	kconf.CurrentContext = hostingClusterContextMock
	if err := kubeconfig.SetHostingClusterContext(kconf, nil); err != nil {
		t.Fatalf("error setupmockcontext: %v", err)
	}
	// Add KubeFlex extension to the test context to make it appear as managed by KubeFlex
	if err := kubeconfig.AssignControlPlaneToContext(kconf, ctxName, certs.GenerateContextName(ctxName)); err != nil {
		t.Fatalf("error assigning control plane to context: %v", err)
	}
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// Setup mock kubeconfig file with context,cluster,authinfo without KubeFlex extension
func setupMockContextWithoutKubeflex(t *testing.T, kubeconfigPath string, ctxName string) {
	kconf := api.NewConfig()
	kconf.Contexts[hostingClusterContextMock] = &api.Context{
		Cluster:  hostingClusterContextMock,
		AuthInfo: hostingClusterContextMock,
	}
	kconf.Contexts[certs.GenerateContextName(ctxName)] = &api.Context{
		Cluster:  certs.GenerateClusterName(ctxName),
		AuthInfo: certs.GenerateAuthInfoAdminName(ctxName),
	}
	kconf.Clusters[certs.GenerateClusterName(ctxName)] = api.NewCluster()
	kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)] = api.NewAuthInfo()
	kconf.CurrentContext = hostingClusterContextMock
	if err := kubeconfig.SetHostingClusterContext(kconf, nil); err != nil {
		t.Fatalf("error setupmockcontext: %v", err)
	}
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// Delete mock kubeconfig file
func teardown(t *testing.T, kubeconfigPath string) {
	if err := os.Remove(kubeconfigPath); err != nil {
		t.Fatalf("failed to teardown: %v", err)
	}
}
