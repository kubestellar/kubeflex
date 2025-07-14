package kubeconfig

import (
	"fmt"
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
	
	// Create a kubeflex config with data
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

	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("Failed to create kubeflex config: %v", err)
	}

	kflexConfig.Extensions.HostingClusterContextName = "test-hosting-cluster"

	runtimeExtension := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("Failed to convert extensions to runtime extension: %v", err)
	}

	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtension

	status, data := CheckHostingClusterContextName(*kconf)

	if status != DiagnosisStatusCritical {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusCritical, status)
	}
	if data == nil {
		t.Errorf("Expected data to not be nil")
	} else if data.HostingClusterContextName != "test-hosting-cluster" {
		t.Errorf("Expected HostingClusterContextName to be 'test-hosting-cluster', got '%s'", data.HostingClusterContextName)
	}

}

func TestCheckHostingClusterContextNameMultiple(t *testing.T) {
	kconf := api.NewConfig()

	// Create global kubeflex extension
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("Failed to create kubeflex config: %v", err)
	}
	kflexConfig.Extensions.HostingClusterContextName = "hosting-cluster-1"

	runtimeExtension := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("Failed to convert extensions to runtime extension: %v", err)
	}
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtension

	// Add cluster and auth info
	cluster := api.NewCluster()
	cluster.Server = "https://example.com:6443"
	kconf.Clusters["example-cluster"] = cluster

	authInfo := api.NewAuthInfo()
	authInfo.Token = "example-token"
	kconf.AuthInfos["example-user"] = authInfo

	// Create first hosting cluster context
	ctx1 := api.NewContext()
	ctx1.Cluster = "example-cluster"
	ctx1.AuthInfo = "example-user"
	ctx1RuntimeExtension := NewRuntimeKubeflexExtension()
	ctx1RuntimeExtension.Data[ExtensionContextsIsHostingCluster] = "true"
	ctx1.Extensions[ExtensionKubeflexKey] = ctx1RuntimeExtension
	kconf.Contexts["hosting-cluster-1"] = ctx1

	// Create second hosting cluster context
	ctx2 := api.NewContext()
	ctx2.Cluster = "example-cluster"
	ctx2.AuthInfo = "example-user"
	ctx2RuntimeExtension := NewRuntimeKubeflexExtension()
	ctx2RuntimeExtension.Data[ExtensionContextsIsHostingCluster] = "true"
	ctx2.Extensions[ExtensionKubeflexKey] = ctx2RuntimeExtension
	kconf.Contexts["hosting-cluster-2"] = ctx2

	status, data := CheckHostingClusterContextName(*kconf)

	if status != DiagnosisStatusWarning {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusWarning, status)
	}
	if data == nil {
		t.Errorf("Expected data to not be nil")
	} else if data.HostingClusterContextName != "hosting-cluster-1" {
		t.Errorf("Expected HostingClusterContextName to be 'hosting-cluster-1', got '%s'", data.HostingClusterContextName)
	}
}

func TestCheckHostingClusterContextNameSingle(t *testing.T) {
	kconf := api.NewConfig()

	// Create global kubeflex extension
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("Failed to create kubeflex config: %v", err)
	}
	kflexConfig.Extensions.HostingClusterContextName = "hosting-cluster"

	runtimeExtension := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("Failed to convert extensions to runtime extension: %v", err)
	}
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtension

	// Add cluster and auth info
	cluster := api.NewCluster()
	cluster.Server = "https://example.com:6443"
	kconf.Clusters["example-cluster"] = cluster

	authInfo := api.NewAuthInfo()
	authInfo.Token = "example-token"
	kconf.AuthInfos["example-user"] = authInfo

	// Create single hosting cluster context
	ctx := api.NewContext()
	ctx.Cluster = "example-cluster"
	ctx.AuthInfo = "example-user"
	ctxRuntimeExtension := NewRuntimeKubeflexExtension()
	ctxRuntimeExtension.Data[ExtensionContextsIsHostingCluster] = "true"
	ctx.Extensions[ExtensionKubeflexKey] = ctxRuntimeExtension
	kconf.Contexts["hosting-cluster"] = ctx

	status, data := CheckHostingClusterContextName(*kconf)

	if status != DiagnosisStatusOK {
		t.Errorf("Expected status '%s', got '%s'", DiagnosisStatusOK, status)
	}
	if data == nil {
		t.Errorf("Expected data to not be nil")
	} else if data.HostingClusterContextName != "hosting-cluster" {
		t.Errorf("Expected HostingClusterContextName to be 'hosting-cluster', got '%s'", data.HostingClusterContextName)
	}
}