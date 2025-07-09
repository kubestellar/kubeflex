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

// Test CheckGlobalKubeflexExtension with no extension present
func TestCheckGlobalKubeflexExtensionCritical(t *testing.T) {
	kconf := api.NewConfig()
	status, data := CheckGlobalKubeflexExtension(*kconf)
	
	if status != "critical" {
		t.Errorf("expected status 'critical', got '%s'", status)
	}
	if data != nil {
		t.Errorf("expected data to be nil, got %v", data)
	}
}

// Test CheckGlobalKubeflexExtension with empty extension
func TestCheckGlobalKubeflexExtensionWarning(t *testing.T) {
	kconf := api.NewConfig()
	
	runtimeExtension := NewRuntimeKubeflexExtension()
	runtimeExtensions, err := runtimeExtension.ConvertExtensionsToRuntimeExtension(runtimeExtension)
	if err != nil {
		t.Fatalf("failed to convert extensions: %v", err)
	}
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtensions[ExtensionKubeflexKey]
	
	status, data := CheckGlobalKubeflexExtension(*kconf)
	
	if status != "warning" {
		t.Errorf("expected status 'warning', got '%s'", status)
	}
	if data != nil {
		t.Errorf("expected data to be nil, got %v", data)
	}
}

// Test CheckGlobalKubeflexExtension with valid extension data
func TestCheckGlobalKubeflexExtensionOk(t *testing.T) {
	kconf := api.NewConfig()
	
	// Create kubeflex config with data
	kflexConfig, err := NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("failed to create kubeflex config: %v", err)
	}
	kflexConfig.Extensions.HostingClusterContextName = "test-hosting-cluster"
	
	// Convert to runtime extension
	runtimeExtension := NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("failed to convert extensions: %v", err)
	}
	
	// Add to kubeconfig
	runtimeExtensions, err := kflexConfig.ParseToKubeconfigExtensions()
	if err != nil {
		t.Fatalf("failed to parse to kubeconfig extensions: %v", err)
	}
	kconf.Extensions[ExtensionKubeflexKey] = runtimeExtensions[ExtensionKubeflexKey]
	
	status, data := CheckGlobalKubeflexExtension(*kconf)
	
	if status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", status)
	}
	if data == nil {
		t.Errorf("expected data to not be nil")
	}
	if data.HostingClusterContextName != "test-hosting-cluster" {
		t.Errorf("expected HostingClusterContextName 'test-hosting-cluster', got '%s'", data.HostingClusterContextName)
	}
}


