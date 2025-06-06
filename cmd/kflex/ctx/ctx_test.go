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
	"fmt"
	"os"

	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigPath string = "./testconfig"
var hostingClusterContextMock = "default"

// Setup mock kubeconfig file with context,cluster,authinfo
func setupMockContext(kubeconfigPath string, ctxName string) error {
	kconf := api.NewConfig()
	kconf.Contexts[certs.GenerateContextName(ctxName)] = &api.Context{
		Cluster:  certs.GenerateClusterName(ctxName),
		AuthInfo: certs.GenerateAuthInfoAdminName(ctxName),
	}
	kconf.Clusters[certs.GenerateClusterName(ctxName)] = api.NewCluster()
	kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)] = api.NewAuthInfo()
	kconf.CurrentContext = hostingClusterContextMock
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
