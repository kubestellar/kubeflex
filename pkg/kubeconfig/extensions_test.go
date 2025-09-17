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

// Unit tests for VerifyControlPlaneOnHostingCluster.
// Covers Critical and Missing scenarios that can be tested without a real cluster.
// OK and deeper Critical cases are tested in the e2e suite.

func TestVerifyControlPlaneOnHostingCluster_MissingControlPlaneName(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	// Create context without kubeflex extension
	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:  "cluster1",
		AuthInfo: "user1",
	}

	result := VerifyControlPlaneOnHostingCluster(*kconf, "ctx1")
	// GetControlPlaneByContextName returns DiagnosisStatusMissing, which is passed as cpName
	// Since it's not empty, it tries to query for a control plane with that name, which fails
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s when control plane name is missing, got %s", DiagnosisStatusCritical, result)
	}
}

func TestVerifyControlPlaneOnHostingCluster_MissingControlPlaneNameWithEmptyExtension(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionContextsIsHostingCluster] = "true"
	// Deliberately not setting ExtensionControlPlaneName

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}

	result := VerifyControlPlaneOnHostingCluster(*kconf, "ctx1")
	if result != DiagnosisStatusMissing {
		t.Errorf("Expected %s when control plane name is empty, got %s", DiagnosisStatusMissing, result)
	}
}

func TestVerifyControlPlaneOnHostingCluster_InvalidKubeconfig(t *testing.T) {
	kconf := api.NewConfig()
	// Create invalid kubeconfig with missing cluster
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	ext := NewRuntimeKubeflexExtension()
	ext.Data[ExtensionContextsIsHostingCluster] = "true"
	ext.Data[ExtensionControlPlaneName] = "test-cp"

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "missing-cluster", // This cluster doesn't exist
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := VerifyControlPlaneOnHostingCluster(*kconf, "ctx1")
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s when kubeconfig is invalid, got %s", DiagnosisStatusCritical, result)
	}
}

func TestVerifyControlPlaneOnHostingCluster_MissingContext(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	result := VerifyControlPlaneOnHostingCluster(*kconf, "nonexistent-ctx")
	// GetControlPlaneByContextName returns DiagnosisStatusMissing, which is passed as cpName
	// Since it's not empty, it tries to query for a control plane with that name, which fails
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s when context doesn't exist, got %s", DiagnosisStatusCritical, result)
	}
}

func TestVerifyControlPlaneOnHostingCluster_BadExtensionData(t *testing.T) {
	kconf := api.NewConfig()
	kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
	kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}

	// Create an extension with invalid data structure
	badExt := &runtime.Unknown{
		Raw: []byte(`invalid-json`),
	}

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: badExt},
	}

	result := VerifyControlPlaneOnHostingCluster(*kconf, "ctx1")
	// GetControlPlaneByContextName returns DiagnosisStatusCritical, which is passed as cpName
	// Since it's not empty, it tries to query for a control plane with that name, which fails
	if result != DiagnosisStatusCritical {
		t.Errorf("Expected %s when extension data is corrupted, got %s", DiagnosisStatusCritical, result)
	}
}

// TestVerifyControlPlaneOnHostingCluster_TableDriven provides comprehensive coverage
// using table-driven tests for better maintainability
func TestVerifyControlPlaneOnHostingCluster_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		setupKubeconfig func() api.Config
		contextName     string
		expectedResult  string
		description     string
	}{
		{
			name: "No extension in context",
			setupKubeconfig: func() api.Config {
				kconf := api.NewConfig()
				kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
				kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
				kconf.Contexts["ctx1"] = &api.Context{
					Cluster:  "cluster1",
					AuthInfo: "user1",
				}
				return *kconf
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusCritical,
			description:    "When context has no kubeflex extension",
		},
		{
			name: "Empty control plane name",
			setupKubeconfig: func() api.Config {
				kconf := api.NewConfig()
				kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
				kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
				ext := NewRuntimeKubeflexExtension()
				ext.Data[ExtensionContextsIsHostingCluster] = "true"
				// ExtensionControlPlaneName is empty
				kconf.Contexts["ctx1"] = &api.Context{
					Cluster:    "cluster1",
					AuthInfo:   "user1",
					Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
				}
				return *kconf
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusMissing,
			description:    "When control plane name is empty in extension",
		},
		{
			name: "Context not found",
			setupKubeconfig: func() api.Config {
				kconf := api.NewConfig()
				kconf.Clusters["cluster1"] = &api.Cluster{Server: "https://example.com:6443"}
				kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
				return *kconf
			},
			contextName:    "nonexistent",
			expectedResult: DiagnosisStatusCritical,
			description:    "When the specified context does not exist",
		},
		{
			name: "Invalid cluster reference",
			setupKubeconfig: func() api.Config {
				kconf := api.NewConfig()
				kconf.AuthInfos["user1"] = &api.AuthInfo{Token: "token"}
				ext := NewRuntimeKubeflexExtension()
				ext.Data[ExtensionControlPlaneName] = "test-cp"
				kconf.Contexts["ctx1"] = &api.Context{
					Cluster:    "nonexistent-cluster",
					AuthInfo:   "user1",
					Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
				}
				kconf.CurrentContext = "ctx1"
				return *kconf
			},
			contextName:    "ctx1",
			expectedResult: DiagnosisStatusCritical,
			description:    "When context references a cluster that doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kconf := tt.setupKubeconfig()
			result := VerifyControlPlaneOnHostingCluster(kconf, tt.contextName)
			if result != tt.expectedResult {
				t.Errorf("Test '%s': expected %s, got %s. Description: %s",
					tt.name, tt.expectedResult, result, tt.description)
			}
		})
	}
}
