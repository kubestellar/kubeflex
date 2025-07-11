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
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
)

// Test delete a context that exists and checks that it is removed from the kubeconfig
func TestDeleteOk(t *testing.T) {
	ctxName := "cptobedeleted"
	setupMockContext(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)

	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxDelete(cp, ctxName, false)
	if err != nil {
		t.Errorf("failed to run 'kflex ctx delete %s': %v", ctxName, err)
	}

	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}

	if _, ok := kconf.Contexts[ctxName]; ok {
		t.Errorf("context '%s' still present in kubeconfig", ctxName)
	}
	if _, ok := kconf.Clusters[certs.GenerateClusterName(ctxName)]; ok {
		t.Errorf("cluster for context '%s' still present", ctxName)
	}
	if _, ok := kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)]; ok {
		t.Errorf("authinfo for context '%s' still present", ctxName)
	}
	if kconf.CurrentContext == ctxName {
		t.Errorf("current context must not be set to deleted context '%s'", ctxName)
	}
}

// Test delete on a non-existent context and ensure kubeconfig is unchanged
func TestDeleteNonExistentContext(t *testing.T) {
	ctxName := "cptobedeleted"
	nonexistent := "nonexistent"
	setupMockContext(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)

	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}
	initialContextCount := len(kconf.Contexts)

	err = ExecuteCtxDelete(cp, nonexistent, false)
	if err == nil {
		t.Errorf("expected deletion to fail for nonexistent context '%s', but it succeeded", nonexistent)
	}

	kconfAfter, _ := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if len(kconfAfter.Contexts) != initialContextCount {
		t.Errorf("context count changed after failed delete: expected %d, got %d", initialContextCount, len(kconfAfter.Contexts))
	}
}

// Test delete a context that is not managed by KubeFlex using --force
func TestDeleteNonKubeflexContext(t *testing.T) {
	ctxName := "cptobedeleted"
	setupMockContextWithoutKubeflex(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)

	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))
	err := ExecuteCtxDelete(cp, ctxName, false, WithForce())
	if err != nil {
		t.Errorf("failed to run 'kflex ctx delete %s' with force: %v", ctxName, err)
	}

	kconf, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		t.Errorf("error loading kubeconfig: %v", err)
	}

	if _, ok := kconf.Contexts[ctxName]; ok {
		t.Errorf("context '%s' still present after forced delete", ctxName)
	}
	if _, ok := kconf.Clusters[certs.GenerateClusterName(ctxName)]; ok {
		t.Errorf("cluster for context '%s' still present", ctxName)
	}
	if _, ok := kconf.AuthInfos[certs.GenerateAuthInfoAdminName(ctxName)]; ok {
		t.Errorf("authinfo for context '%s' still present", ctxName)
	}
	if kconf.CurrentContext == ctxName {
		t.Errorf("current context must not be set to deleted context '%s'", ctxName)
	}
}

// Test that deleting a non-KubeFlex-managed context prompts a guard (confirmation)
func TestDeleteNonKubeflexContext_PromptsGuard(t *testing.T) {
	ctxName := "nonkubeflexguard"
	setupMockContextWithoutKubeflex(t, kubeconfigPath, ctxName)
	defer teardown(t, kubeconfigPath)

	cp := common.NewCP(kubeconfigPath, common.WithName(ctxName))

	// Simulate user input: "n" (do not confirm)
	r, w, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()
	go func() {
		w.Write([]byte("n\n"))
		w.Close()
	}()

	// Capture output
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() { os.Stdout = oldStdout }()

	err := ExecuteCtxDelete(cp, ctxName, false)
	wOut.Close()
	var buf bytes.Buffer
	buf.ReadFrom(rOut)
	output := buf.String()
	if !strings.Contains(output, "Warning: Context") || !strings.Contains(output, "Are you sure you want to delete this context?") {
		t.Errorf("Expected guard prompt for non-KubeFlex context, got output: %s", output)
	}
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}
