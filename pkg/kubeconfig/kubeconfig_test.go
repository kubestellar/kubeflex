package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd/api"
	"testing"
)

// Setup and teardown helpers
func setupTestKubeconfigWithContext(ctxName string) *api.Config {
	kconf := api.NewConfig()
	kconf.Contexts[ctxName] = &api.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-user",
	}
	return kconf
}

func teardownTestKubeconfig(kconf *api.Config, ctxName string) {
	delete(kconf.Contexts, ctxName)
}

func TestAssignControlPlaneToContext_Positive(t *testing.T) {
	ctxName := "test-context"
	cpName := "test-controlplane"
	kconf := setupTestKubeconfigWithContext(ctxName)
	defer teardownTestKubeconfig(kconf, ctxName)

	err := AssignControlPlaneToContext(kconf, cpName, ctxName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := kconf.Contexts[ctxName].Extensions[ExtensionKubeflexKey]
	if !ok {
		t.Fatalf("expected kubeflex extension to be set in context")
	}

	kflexConfig, err := NewKubeflexContextConfig(*kconf, ctxName)
	if err != nil {
		t.Fatalf("expected no error creating kubeflex context config, got: %v", err)
	}
	runtimeExt := NewRuntimeKubeflexExtension()
	err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExt)
	if err != nil {
		t.Fatalf("expected no error converting extensions to runtime extension, got: %v", err)
	}

	if got := runtimeExt.Data[ExtensionControlPlaneName]; got != cpName {
		t.Errorf("expected controlplane-name to be '%s', got '%s'", cpName, got)
	}
}

func TestAssignControlPlaneToContext_Negative_ContextDoesNotExist(t *testing.T) {
	ctxName := "nonexistent-context"
	cpName := "test-controlplane"
	kconf := api.NewConfig()

	err := AssignControlPlaneToContext(kconf, cpName, ctxName)
	if err == nil {
		t.Fatalf("expected error for nonexistent context, got nil")
	}
}
