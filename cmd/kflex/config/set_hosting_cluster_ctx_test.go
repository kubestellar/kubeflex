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

package config

import (
	"os"
	"path"
	"testing"

	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigPath string = "./testconfig"
var hostingClusterContextMock = "kind-kubeflex"

// setupKubeconfig creates a mock kubeconfig file with context, cluster, and authinfo
func setupKubeconfig(t *testing.T, kubeconfigPath string, ctxName string) {
	kconf := api.NewConfig()
	// hosting cluster context entry
	kconf.Contexts[hostingClusterContextMock] = &api.Context{
		Cluster:  hostingClusterContextMock,
		AuthInfo: hostingClusterContextMock,
	}
	kconf.Clusters[hostingClusterContextMock] = api.NewCluster()
	kconf.AuthInfos[hostingClusterContextMock] = api.NewAuthInfo()
	// ctxName entry
	kconf.Contexts[certs.GenerateContextName(ctxName)] = &api.Context{
		Cluster:  certs.GenerateClusterName(ctxName),
		AuthInfo: certs.GenerateAuthInfoAdminName(ctxName),
	}
	kconf.Clusters[certs.GenerateClusterName(ctxName)] = api.NewCluster()
	kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)] = api.NewAuthInfo()
	kconf.CurrentContext = hostingClusterContextMock
	if err := kubeconfig.SetHostingClusterContext(kconf, nil); err != nil {
		t.Fatalf("error setup mock kubeconfig: %v", err)
	}
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// setupInvalidKubeconfig creates an invalid mock kubeconfig file
func setupInvalidKubeconfig(t *testing.T, kubeconfigPath string, ctxName string) {
	setupKubeconfig(t, kubeconfigPath, ctxName+"-makeitinvalid")
}

// Delete mock kubeconfig file
func teardown(t *testing.T, kubeconfigPath string) {
	if err := os.Remove(kubeconfigPath); err != nil {
		t.Fatalf("failed to teardown: %v", err)
	}
}

// Test set hosting cluster ctx to $ctxName is successful
// when a kubeconfig is valid meaning has $ctxName as entry
// to contexts
func TestSetHostingClusterCtx_ValidKubeconfig(t *testing.T) {
	ctxName := "cp1"
	kubeconfigPath = path.Join(t.TempDir(), kubeconfigPath)
	setupKubeconfig(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)
	// Execute command - change hosting cluster ctx name to $ctxName
	err := ExecuteSetHostingClusterCtx(kubeconfigPath, ctxName)
	if err != nil {
		t.Errorf("execute set hosting cluster ctx command failed: %v", err)
	}
	// Checking values
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}
	kflexConfig, err := kubeconfig.NewKubeflexConfig(*kconf)
	if err != nil {
		t.Errorf("error creating kflexConfig: %v", err)
	}
	kflexContextConfig, err := kubeconfig.NewKubeflexContextConfig(*kconf, ctxName)
	if err != nil {
		t.Errorf("error creating kflexContextConfig: %v", err)
	}
	if kflexConfig.Extensions.HostingClusterContextName != ctxName {
		t.Errorf("hosting cluster context name must be changed from '%s' to '%s', not '%s'", hostingClusterContextMock, ctxName, kflexConfig.Extensions.HostingClusterContextName)
	}
	if kflexContextConfig.Extensions.IsHostingClusterContext != "true" {
		t.Errorf("hosting cluster context must indicates to be hosting cluster using 'true' not '%s'", kflexContextConfig.Extensions.IsHostingClusterContext)
	}
}

// Test set hosting cluster ctx fails to change hosting cluster context name to $ctxName
// when a kubeconfig does not have context entry of $ctxName
func TestSetHostingClusterCtx_InvalidKubeconfig(t *testing.T) {
	ctxName := "cp1"
	kubeconfigPath = path.Join(t.TempDir(), kubeconfigPath)
	setupInvalidKubeconfig(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)
	// Execute command - change hosting cluster ctx name to $ctxName should failed
	err := ExecuteSetHostingClusterCtx(kubeconfigPath, ctxName)
	if err == nil {
		t.Errorf("execute set hosting cluster ctx command failed: %v", err)
	}
	// Checking values
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}
	kflexConfig, err := kubeconfig.NewKubeflexConfig(*kconf)
	if err != nil {
		t.Errorf("error creating kflexConfig: %v", err)
	}
	// hosting cluster context name should remain unchanged
	if kflexConfig.Extensions.HostingClusterContextName != hostingClusterContextMock {
		t.Errorf("hosting cluster context name must be changed from '%s' to '%s', not '%s'", hostingClusterContextMock, ctxName, kflexConfig.Extensions.HostingClusterContextName)
	}
}
