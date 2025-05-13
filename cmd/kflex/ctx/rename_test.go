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
	"testing"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var kubeconfigPath string = clientcmd.RecommendedHomeFile

func setupMockContext(kubeconfigPath string, ctxName string) error {
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	kconf.Contexts[certs.GenerateContextName(ctxName)] = &api.Context{
		Cluster:  certs.GenerateClusterName(ctxName),
		AuthInfo: certs.GenerateAuthInfoAdminName(ctxName),
	}
	if err = kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	return nil
}

func teardown(kubeconfigPath string, ctxName string) error {
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	if err = kubeconfig.DeleteContext(kconf, ctxName); err != nil {
		return fmt.Errorf("cannot delete context %s from kubeconfig: %v", ctxName, err)
	}
	kconf.CurrentContext, err = kubeconfig.GetHostingClusterContext(kconf)
	if err != nil {
		return err
	}
	if err = kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		return fmt.Errorf("error writing kubeconfig: %v", err)
	}
	return nil
}

func TestRenameOk(t *testing.T) {
	// Mock data
	ctxName := "testcp"
	expected := "testcp-renamed"
	setupMockContext(kubeconfigPath, ctxName)
	defer teardown(kubeconfigPath, expected)

	// Start test
	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxRename(cp, ctxName, expected, false)
	if err != nil {
		t.Errorf("fail to execute rename: %v", err)
	}
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	fmt.Printf("Current context is %s\n", kconf.CurrentContext)
	if err != nil {
		t.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	if _, ok := kconf.Contexts[certs.GenerateContextName(expected)]; !ok {
		t.Errorf("control plane context name must be '%s' not '%s'", certs.GenerateContextName(expected), certs.GenerateContextName(cp.Name))
	}
	if kconf.Contexts[certs.GenerateContextName(expected)].Cluster != certs.GenerateClusterName(expected) {
		t.Errorf("control plane cluster name must be '%s' not '%s'", certs.GenerateClusterName(expected), kconf.Contexts[certs.GenerateContextName(expected)].Cluster)
	}
	if kconf.Contexts[certs.GenerateContextName(expected)].AuthInfo != certs.GenerateAuthInfoAdminName(expected) {
		t.Errorf("control plane user name must be '%s' not '%s'", certs.GenerateAuthInfoAdminName(expected), kconf.Contexts[certs.GenerateContextName(expected)].AuthInfo)
	}
	fmt.Printf("Current context is %s\n", kconf.CurrentContext)
	// Check current context
	if hostingClusterCtx, err := kubeconfig.GetHostingClusterContext(kconf); err != nil || kconf.CurrentContext != hostingClusterCtx {
		t.Errorf("control plane current context must be '%s' not '%s'", hostingClusterCtx, kconf.CurrentContext)
	}
}

func TestRenameThenSwitchOk(t *testing.T) {
	// Mock data
	ctxName := "testcp"
	expected := "testcp-renamed"
	setupMockContext(kubeconfigPath, ctxName)
	defer teardown(kubeconfigPath, expected)

	// Start test
	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxRename(cp, ctxName, expected, true) // Enable switch
	if err != nil {
		t.Errorf("fail to execute rename: %v", err)
	}
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	if _, ok := kconf.Contexts[certs.GenerateContextName(expected)]; !ok {
		t.Errorf("control plane context name must be '%s' not '%s'", certs.GenerateContextName(expected), certs.GenerateContextName(cp.Name))
	}
	if kconf.Contexts[certs.GenerateContextName(expected)].Cluster != certs.GenerateClusterName(expected) {
		t.Errorf("control plane cluster name must be '%s' not '%s'", certs.GenerateClusterName(expected), kconf.Contexts[certs.GenerateContextName(expected)].Cluster)
	}
	if kconf.Contexts[certs.GenerateContextName(expected)].AuthInfo != certs.GenerateAuthInfoAdminName(expected) {
		t.Errorf("control plane user name must be '%s' not '%s'", certs.GenerateAuthInfoAdminName(expected), kconf.Contexts[certs.GenerateContextName(expected)].AuthInfo)
	}
	// Check current context
	if kconf.CurrentContext != certs.GenerateContextName(expected) {
		t.Errorf("control plane current context must be '%s' not '%s'", certs.GenerateContextName(expected), kconf.CurrentContext)
	}
}

func TestRenameNonExistentContext(t *testing.T) {
	// Mock data
	ctxName := "nonexistent"
	expected := "testcp-renamed"

	// Start test
	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxRename(cp, ctxName, expected, false)
	if err == nil {
		t.Errorf("rename command has been executed without error but should have return an error: %v", err)
	}
	fmt.Printf("expected error: %v\n", err)
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("no kubeconfig context for %s was found, cannot load: %v", ctxName, err)
	}
	if _, ok := kconf.Contexts[certs.GenerateContextName(ctxName)]; ok {
		t.Errorf("control plane should not have '%s' as context", certs.GenerateContextName(ctxName))
	}
	if _, ok := kconf.Contexts[certs.GenerateContextName(expected)]; ok {
		t.Errorf("control plane should not have '%s' as context", certs.GenerateContextName(expected))
	}
}
