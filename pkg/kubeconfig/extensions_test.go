package kubeconfig

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

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
	// Missing control-plane name

	kconf.Contexts["ctx1"] = &api.Context{
		Cluster:    "cluster1",
		AuthInfo:   "user1",
		Extensions: map[string]runtime.Object{ExtensionKubeflexKey: ext},
	}
	kconf.CurrentContext = "ctx1"

	result := CheckContextScopeKubeflexExtensionSet(*kconf)
	if result != DiagnosisStatusWarning {
		t.Errorf("Expected %s, got %s", DiagnosisStatusWarning, result)
	}
}
