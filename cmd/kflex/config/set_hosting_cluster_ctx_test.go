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
	"fmt"
	"os"
	"testing"

	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigPath string = "./testconfig"
var hostingClusterContextMock = "kind-kubeflex"

// Setup mock kubeconfig file with context,cluster,authinfo
func setupKubeconfig(kubeconfigPath string, ctxName string) error {
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
	fmt.Printf("kconf: %v\n", kconf)
	if err := kubeconfig.SetHostingClusterContext(kconf, nil); err != nil {
		return fmt.Errorf("error setupmockcontext: %v", err)
	}
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	return nil
}

// Delete mock kubeconfig file
func teardown(kubeconfigPath string) error {
	return os.Remove(kubeconfigPath)
}

func TestSetHostingClusterCtx_ValidKubeconfig(t *testing.T) {
	ctxName := "cp1"
	setupKubeconfig(kubeconfigPath, ctxName)
	defer teardown(kubeconfigPath)
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
