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

package config

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/kubestellar/kubeflex/pkg/kubeconfig"
	"k8s.io/client-go/tools/clientcmd/api"
)

// setupKubeconfigCritical creates a kubeconfig without kubeflex extension
func setupKubeconfigCritical(t *testing.T, kubeconfigPath string) {
	kconf := api.NewConfig()
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// setupKubeconfigWarning creates a kubeconfig with empty kubeflex extension
func setupKubeconfigWarning(t *testing.T, kubeconfigPath string) {
	kconf := api.NewConfig()
	runtimeExtension := kubeconfig.NewRuntimeKubeflexExtension()
	kconf.Extensions[kubeconfig.ExtensionKubeflexKey] = runtimeExtension
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// setupKubeconfigOk creates a kubeconfig with valid kubeflex extension
func setupKubeconfigOk(t *testing.T, kubeconfigPath string) {
	kconf := api.NewConfig()
	kflexConfig, err := kubeconfig.NewKubeflexConfig(*kconf)
	if err != nil {
		t.Fatalf("error creating kubeflex config: %v", err)
	}
	kflexConfig.Extensions.HostingClusterContextName = "test-hosting-cluster"
	
	runtimeExtension := kubeconfig.NewRuntimeKubeflexExtension()
	if err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExtension); err != nil {
		t.Fatalf("error converting extensions: %v", err)
	}
	
	runtimeExtensions, err := kflexConfig.ParseToKubeconfigExtensions()
	if err != nil {
		t.Fatalf("error parsing to kubeconfig extensions: %v", err)
	}
	kconf.Extensions[kubeconfig.ExtensionKubeflexKey] = runtimeExtensions[kubeconfig.ExtensionKubeflexKey]
	
	if err := kubeconfig.WriteKubeconfig(kubeconfigPath, kconf); err != nil {
		t.Fatalf("error writing kubeconfig: %v", err)
	}
}

// Test diagnose with critical status (no extension)
func TestExecuteDiagnose_Critical(t *testing.T) {
	kubeconfigPath := path.Join(t.TempDir(), "testconfig")
	setupKubeconfigCritical(t, kubeconfigPath)
	
	err := ExecuteDiagnose(kubeconfigPath, false)
	if err != nil {
		t.Errorf("ExecuteDiagnose failed: %v", err)
	}
}

// Test diagnose with warning status (empty extension)
func TestExecuteDiagnose_Warning(t *testing.T) {
	kubeconfigPath := path.Join(t.TempDir(), "testconfig")
	setupKubeconfigWarning(t, kubeconfigPath)
	
	err := ExecuteDiagnose(kubeconfigPath, false)
	if err != nil {
		t.Errorf("ExecuteDiagnose failed: %v", err)
	}
}

// Test diagnose with ok status (valid extension)
func TestExecuteDiagnose_Ok(t *testing.T) {
	kubeconfigPath := path.Join(t.TempDir(), "testconfig")
	setupKubeconfigOk(t, kubeconfigPath)
	
	err := ExecuteDiagnose(kubeconfigPath, false)
	if err != nil {
		t.Errorf("ExecuteDiagnose failed: %v", err)
	}
}

// Test diagnose with JSON output
func TestExecuteDiagnose_JSON(t *testing.T) {
	kubeconfigPath := path.Join(t.TempDir(), "testconfig")
	setupKubeconfigOk(t, kubeconfigPath)
	
	err := ExecuteDiagnose(kubeconfigPath, true)
	if err != nil {
		t.Errorf("ExecuteDiagnose failed: %v", err)
	}
}

// Test diagnose with invalid kubeconfig file
func TestExecuteDiagnose_InvalidFile(t *testing.T) {
	err := ExecuteDiagnose("/non/existent/file", false)
	if err == nil {
		t.Errorf("expected error for non-existent kubeconfig file")
	}
}

func TestDiagnosisResultJSON(t *testing.T) {
	result := DiagnosisResult{
		Status:  "ok",
		Message: "Global kubeflex extension is present and properly configured",
		Data: &kubeconfig.KubeflexExtensions{
			HostingClusterContextName: "test-cluster",
		},
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Errorf("failed to marshal DiagnosisResult to JSON: %v", err)
	}

	var unmarshaled DiagnosisResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("failed to unmarshal DiagnosisResult from JSON: %v", err)
	}

	if unmarshaled.Status != result.Status {
		t.Errorf("expected status %s, got %s", result.Status, unmarshaled.Status)
	}
	if unmarshaled.Message != result.Message {
		t.Errorf("expected message %s, got %s", result.Message, unmarshaled.Message)
	}
	if unmarshaled.Data == nil || unmarshaled.Data.HostingClusterContextName != result.Data.HostingClusterContextName {
		t.Errorf("expected hosting cluster context %s, got %s", result.Data.HostingClusterContextName, unmarshaled.Data.HostingClusterContextName)
	}
}
