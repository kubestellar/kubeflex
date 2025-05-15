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
	"testing"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
)

// Test delete a context that exist and checks that it is removed
// from the kubeconfig
func TestDeleteOk(t *testing.T) {
	ctxName := "cptobedeleted"
	setupMockContext(kubeconfigPath, ctxName)
	defer teardown(kubeconfigPath)

	// Start test
	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxDelete(cp, ctxName, false)
	if err != nil {
		t.Errorf("failed to run 'kflex ctx delete %s': %v", ctxName, err)
	}
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}
	clusterName := certs.GenerateClusterName(ctxName)
	authInfoName := certs.GenerateAuthInfoAdminName(ctxName)
	if c, ok := kconf.Contexts[ctxName]; ok {
		t.Errorf("context '%v' still present in kubeconfig", c)
	}
	if c, ok := kconf.Clusters[clusterName]; ok {
		t.Errorf("cluster '%v' still present in kubeconfig", c)
	}
	if c, ok := kconf.AuthInfos[authInfoName]; ok {
		t.Errorf("user '%v' still present in kubeconfig", c)
	}
	if kconf.CurrentContext == ctxName {
		t.Errorf("current context must not be set as the deleted context %s", ctxName)
	}
}

// Test delete on non-existent context and checks that the kubeconfig is unchanged
func TestDeleteNonExistentContext(t *testing.T) {
	ctxName := "cptobedeleted"
	noneCtxName := "none"
	setupMockContext(kubeconfigPath, ctxName)
	defer teardown(kubeconfigPath)

	// Start test
	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}
	nCtx := len(kconf.Contexts)
	err = ExecuteCtxDelete(cp, noneCtxName, false)
	if err == nil {
		t.Errorf("expect ExecuteCtxDelete to fail but it succeeded")
	}
	if nCtx != len(kconf.Contexts) {
		t.Errorf("expect ExecuteCtxDelete to not delete any context but it did")
	}
}
