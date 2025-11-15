package kubeconfig

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/tools/clientcmd/api"
)

const hostingClusterContextName = "kind-kubeflex"
const hostingClusterContextNameExistent = "kind-legacy-kubeflex"

// Setup mock values to KubeflexConfig
func SetupMockKubeflexConfig(receiver **KubeflexConfig) (err error) {
	kconf := api.NewConfig()
	if *receiver == nil {
		fmt.Println("setup mock with new values")
		*receiver, err = NewKubeflexConfig(*kconf)
		if err != nil {
			return err
		}
		(*receiver).Extensions.HostingClusterContextName = hostingClusterContextName
	} else {
		// Define extensions.kubeflex mock values
		fmt.Println("setup mock with existent values")
		(*receiver).Extensions.HostingClusterContextName = hostingClusterContextNameExistent
	}
	return nil
}

// Test KubeflexConfig has correct mock values
func TestKubeflexConfig(t *testing.T) {
	var kflexConfig *KubeflexConfig
	err := SetupMockKubeflexConfig(&kflexConfig)
	if err != nil {
		t.Errorf("fail to setup kubeflex config")
	}
	if kflexConfig.Extensions.HostingClusterContextName != hostingClusterContextName {
		t.Errorf("fail to setup kubeflex config as HostingClusterContextName is not '%s'", hostingClusterContextName)
	}
}

// Test KubeflexConfig has correct mock values if it is not null
func TestKubeflexConfigWithExistentValues(t *testing.T) {
	kflexConfig, err := NewKubeflexConfig(*api.NewConfig())
	if err != nil {
		t.Errorf("fail to create new kubeflex config")
	}
	err = SetupMockKubeflexConfig(&kflexConfig)
	if err != nil {
		t.Errorf("fail to setup kubeflex config")
	}
	if kflexConfig.Extensions.HostingClusterContextName != hostingClusterContextNameExistent {
		t.Errorf("fail to setup kubeflex config as HostingClusterContextName is not '%s'", hostingClusterContextNameExistent)
	}
}

// Test conversion of KubeflexConfig to Kubeconfig extensions data
func TestKubeflexConfigWrittenAsKubeConfig(t *testing.T) {
	kflexConfig, _ := NewKubeflexConfig(*api.NewConfig())
	err := SetupMockKubeflexConfig(&kflexConfig)
	if err != nil {
		t.Errorf("fail to setup")
	}
	fmt.Printf("kflexConfig: extensions: %v\n", kflexConfig.Extensions)
	runtimeKflex := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeKflex); err != nil {
		t.Errorf("fail to convert extensions to runtime extension")
	}
	fmt.Printf("runtimeKflex metadata: %v\n", runtimeKflex.ObjectMeta)
	fmt.Printf("runtimeKflex data: %v\n", runtimeKflex.Data)
	if v, ok := runtimeKflex.Data[ExtensionHostingClusterContextName]; !ok || v != hostingClusterContextNameExistent {
		t.Errorf("fail to setup kubeflex config as HostingClusterContextName is not '%s': value is %s", hostingClusterContextNameExistent, v)
	}
}

// Test CheckGlobalKubeflexExtension when extension is not set
func TestCheckGlobalKubeflexExtensionNotSet(t *testing.T) {
	kconf := api.NewConfig()
	status, data := CheckGlobalKubeflexExtension(*kconf)
	if status != DiagnosisStatusCritical {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusCritical, status)
	}
	if data != nil {
		t.Errorf("Expected data to be nil, got %v", data)
	}
}

// Test CheckGlobalKubeflexExtension when extension is set but empty
func TestCheckGlobalKubeflexExtensionEmpty(t *testing.T) {
	kconf := api.NewConfig()

	// Create an empty runtime extension
	runtimeExtension := NewRuntimeKubeflexExtension()

	// Don't add any data to the extension
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtension
	status, data := CheckGlobalKubeflexExtension(*kconf)
	if status != DiagnosisStatusWarning {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusWarning, status)
	}
	if data != nil {
		t.Errorf("Expected data to be nil, got %v", data)
	}
}

// Test CheckGlobalKubeflexExtension when extension is set with valid data
func TestCheckGlobalKubeflexExtensionWithData(t *testing.T) {
	kconf := api.NewConfig()
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("Failed to create kubeflex config: %v", err)
	}
	kflexConfig.Extensions.HostingClusterContextName = "test-hosting-cluster"

	// Convert to runtime extension and add to kubeconfig
	runtimeExtension := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("Failed to convert extensions to runtime extension: %v", err)
	}
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtension

	status, data := CheckGlobalKubeflexExtension(*kconf)
	if status != DiagnosisStatusOK {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusOK, status)
	}
	if data == nil {
		t.Errorf("Expected data to not be nil")
	} else if data.HostingClusterContextName != "test-hosting-cluster" {
		t.Errorf("Expected HostingClusterContextName to be 'test-hosting-cluster', got '%s'", data.HostingClusterContextName)
	}
}

func TestCheckHostingClusterContextNameNone(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionContextsIsHostingCluster] = "false"

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckHostingClusterContextName(*kconf)
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s, got %s", DiagnosisStatusCritical, result)
	}
}

func TestCheckHostingClusterContextNameMultiple(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.Clusters["cluster2"] = &api.Cluster{Server: "https://example.org:6443"}

	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token1"}
	kconf.AuthInfos["user2"] = &api.AuthInfo{Token: "token2"}

	ext1 := NewRuntimeKubeflexExtension()
	ext1.Data[ExtensionContextsIsHostingCluster] = "true"
	ext2 := NewRuntimeKubeflexExtension()
	ext2.Data[ExtensionContextsIsHostingCluster] = "true"

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext1},
	}
	kconf.Contexts["ctx2"] = &api.Context{
		Cluster:    "cluster2",
		AuthInfo:   "user2",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext2},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckHostingClusterContextName(*kconf)
	if result != DiagnosisStatusWarning {
		t.Errorf("Expected %s, got %s", DiagnosisStatusWarning, result)
	}
}

func TestCheckHostingClusterContextNameSingle(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionContextsIsHostingCluster] = "true"

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckHostingClusterContextName(*kconf)
	if result != DiagnosisStatusOK {
		t.Errorf("Expected %s, got %s", DiagnosisStatusOK, result)
	}
}

func TestCheckContextScopeKubeflexExtensionSetNoKubeflexExtensions(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:  "cluster1",
		AuthInfo: "user1",
	}
	kconf.CurrentContext = "ctx1"

	result := CheckContextScopeKubeflexExtensionSet(*kconf, "ctx1")
	if result != DiagnosisStatusMissing {
		t.Errorf("Expected %s, got %s", DiagnosisStatusMissing, result)
	}
}

func TestCheckContextScopeKubeflexExtensionSetNoData(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data = nil

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckContextScopeKubeflexExtensionSet(*kconf, "ctx1")
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s, got %s", DiagnosisStatusCritical, result)
	}
}

func TestCheckContextScopeKubeflexExtensionSetPartialData(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionContextsIsHostingCluster] = "true"

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckContextScopeKubeflexExtensionSet(*kconf, "ctx1")
	if result != DiagnosisStatusWarning {
		t.Errorf("Expected %s, got %s", DiagnosisStatusWarning, result)
	}
}

func TestCheckExtensionInitialContextNameSetFalse(t *testing.T) {
	kconf := api.NewConfig()
	ext := NewRuntimeKubeflexExtension()

	kconf.Extensions = map[string]runtime.Object{
		ExtensionKubeflexKey: ext,
	}

	status := CheckExtensionInitialContextNameSet(*kconf)
	if status != DiagnosisStatusWarning {
		t.Errorf("Expected %s when ExtensionInitialContextName is not set, got %s", DiagnosisStatusWarning, status)
	}
}

func TestCheckExtensionInitialContextNameSetTrue(t *testing.T) {
	kconf := api.NewConfig()

	// create a valid context and cluster so validation succeeds
	kconf.Contexts["kind-kubeflex"] = &api.Context{Cluster: "cluster1", AuthInfo: "user1"}
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionInitialContextName] = "kind-kubeflex"

	kconf.Extensions = map[string]runtime.Object{
		ExtensionKubeflexKey: ext,
	}

	status := CheckExtensionInitialContextNameSet(*kconf)
	if status != DiagnosisStatusOK {
		t.Errorf("Expected %s when ExtensionInitialContextName is set, got %s", DiagnosisStatusOK, status)
	}
}

// When the extension points to a non-existent context it should be treated as not set
func TestCheckExtensionInitialContextNameSetNonExistentContext(t *testing.T) {
	kconf := api.NewConfig()

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionInitialContextName] = "does-not-exist"

	kconf.Extensions = map[string]runtime.Object{
		ExtensionKubeflexKey: ext,
	}

	status := CheckExtensionInitialContextNameSet(*kconf)
	if status != DiagnosisStatusWarning {
		t.Errorf("Expected %s when ExtensionInitialContextName points to non-existent context, got %s", DiagnosisStatusWarning, status)
	}
}

// When the extension points to a context whose cluster has no server, treat as not set
func TestCheckExtensionInitialContextNameSetMissingClusterServer(t *testing.T) {
	kconf := api.NewConfig()

	// create context that references a cluster without server
	kconf.Contexts["ctx-no-server"] = &api.Context{Cluster: "cluster-no-server", AuthInfo: "user1"}
	kconf.Clusters["cluster-no-server"] = &api.Cluster{Server: ""}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionInitialContextName] = "ctx-no-server"

	kconf.Extensions = map[string]runtime.Object{
		ExtensionKubeflexKey: ext,
	}

	status := CheckExtensionInitialContextNameSet(*kconf)
	if status != DiagnosisStatusWarning {
		t.Errorf("Expected %s when ExtensionInitialContextName points to context with missing cluster server, got %s", DiagnosisStatusWarning, status)
	}
}

// TestVerifyControlPlaneOnHostingCluster_TableDriven provides comprehensive coverage
// using table-driven tests for better maintainability and refactored per review.
// Note: DiagnosisStatusOK cases require a real cluster and are tested in the e2e suite.
func TestVerifyControlPlaneOnHostingCluster_TableDriven(t *testing.T) {
	baseConfig := func() api.Config {
		kconf := api.NewConfig()
		kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
		kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
		kconf.Contexts["ctx1"] = &api.Context{
			Cluster:  "cluster1",
			AuthInfo: "user1",
		}
		kconf.CurrentContext = "ctx1"
		return *kconf
	}

	tests := []struct {
		name            string
		tweakKubeconfig func(*api.Config)
		contextName     string
		expectedResult  string
		description     string
	}{
		{
			name: "No extension in context",
			tweakKubeconfig: func(kconf *api.Config) {
				// Leave context with no kubeflex extension
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusCritical,
			description:    "When context has no kubeflex extension",
		},
		{
			name: "Empty control plane name",
			tweakKubeconfig: func(kconf *api.Config) {
				ext := NewRuntimeKubeflexExtension()
				ext.Data[ExtensionContextsIsHostingCluster] = "true"
				// Deliberately not setting ExtensionControlPlaneName
				kconf.Contexts["ctx1"].Extensions = map[string]runtime.Object{ExtensionKubeflexKey: ext}
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusMissing,
			description:    "When control plane name is empty in extension",
		},
		{
			name: "Invalid cluster reference",
			tweakKubeconfig: func(kconf *api.Config) {
				ext := NewRuntimeKubeflexExtension()
				ext.Data[ExtensionContextsIsHostingCluster] = "true"
				ext.Data[ExtensionControlPlaneName] = "test-cp"
				kconf.Contexts["ctx1"].Cluster = "missing-cluster"
				kconf.Contexts["ctx1"].Extensions = map[string]runtime.Object{ExtensionKubeflexKey: ext}
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusCritical,
			description:    "When context references a cluster that doesn't exist",
		},
		{
			name: "Corrupted extension data",
			tweakKubeconfig: func(kconf *api.Config) {
				badExt := &runtime.Unknown{
					Raw: []byte(`invalid-json`),
				}
				kconf.Contexts["ctx1"].Extensions = map[string]runtime.Object{ExtensionKubeflexKey: badExt}
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusCritical,
			description:    "When extension data is corrupted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kconf := baseConfig()
			tt.tweakKubeconfig(&kconf)
			result := VerifyControlPlaneOnHostingCluster(kconf, tt.contextName)
			if result != tt.expectedResult {
				t.Errorf("Test '%s': expected %s, got %s. %s",
					tt.name, tt.expectedResult, result, tt.description)
			}
		})
	}
}
