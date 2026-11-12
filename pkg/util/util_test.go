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

package util

import (
	"os"
	"testing"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGenerateNamespaceFromControlPlaneName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		expected string
	}{
		{
			name:     "simple control plane name",
			cpName:   "my-cp",
			expected: "my-cp-system",
		},
		{
			name:     "empty control plane name",
			cpName:   "",
			expected: "-system",
		},
		{
			name:     "control plane name with special chars",
			cpName:   "test-cp-123",
			expected: "test-cp-123-system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateNamespaceFromControlPlaneName(tt.cpName)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestControlPlaneNameFromNamespace(t *testing.T) {
	tests := []struct {
		name        string
		nsName      string
		expected    string
		expectError bool
	}{
		{
			name:        "valid namespace with suffix",
			nsName:      "my-cp-system",
			expected:    "my-cp",
			expectError: false,
		},
		{
			name:        "namespace without suffix",
			nsName:      "my-namespace",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty namespace",
			nsName:      "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "namespace with only suffix",
			nsName:      "-system",
			expected:    "",
			expectError: true,
		},
		{
			name:        "namespace with suffix in middle",
			nsName:      "my-system-cp",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ControlPlaneNameFromNamespace(tt.nsName)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateDevLocalDNSName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		domain   string
		expected string
	}{
		{
			name:     "standard case",
			cpName:   "my-cp",
			domain:   "example.com",
			expected: "my-cp.example.com",
		},
		{
			name:     "empty domain",
			cpName:   "my-cp",
			domain:   "",
			expected: "my-cp.",
		},
		{
			name:     "empty control plane name",
			cpName:   "",
			domain:   "example.com",
			expected: ".example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDevLocalDNSName(tt.cpName, tt.domain)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateHostedDNSName(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		svcName   string
		expected  []string
	}{
		{
			name:      "standard case",
			namespace: "test-ns",
			svcName:   "my-service",
			expected: []string{
				"my-service.test-ns",
				"my-service.test-ns.svc",
				"my-service.test-ns.svc.cluster",
				"my-service.test-ns.svc.cluster.local",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHostedDNSName(tt.namespace, tt.svcName)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d DNS names, got %d", len(tt.expected), len(result))
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("at index %d: expected %q, got %q", i, exp, result[i])
				}
			}
		})
	}
}

func TestGenerateOperatorDeploymentName(t *testing.T) {
	expected := "kubeflex-controller-manager"
	result := GenerateOperatorDeploymentName()
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseVersionNumber(t *testing.T) {
	tests := []struct {
		name          string
		versionString string
		expected      string
	}{
		{
			name:          "standard version",
			versionString: "v1.28.3",
			expected:      "28",
		},
		{
			name:          "simple version",
			versionString: "1.27",
			expected:      "27",
		},
		{
			name:          "single part version",
			versionString: "1",
			expected:      "1",
		},
		{
			name:          "empty version",
			versionString: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseVersionNumber(tt.versionString)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetKubeconfSecretNameByControlPlaneType(t *testing.T) {
	tests := []struct {
		name             string
		controlPlaneType string
		expected         string
	}{
		{
			name:             "K8S type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK8S),
			expected:         AdminConfSecret,
		},
		{
			name:             "OCM type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeOCM),
			expected:         OCMKubeConfigSecret,
		},
		{
			name:             "VCluster type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeVCluster),
			expected:         VClusterKubeConfigSecret,
		},
		{
			name:             "K3s type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK3s),
			expected:         K3sKubeConfigSecret,
		},
		{
			name:             "unknown type",
			controlPlaneType: "unknown",
			expected:         AdminConfSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKubeconfSecretNameByControlPlaneType(tt.controlPlaneType)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetKubeconfSecretKeyNameByControlPlaneType(t *testing.T) {
	tests := []struct {
		name             string
		controlPlaneType string
		expected         string
	}{
		{
			name:             "K8S type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK8S),
			expected:         KubeconfigSecretKeyDefault,
		},
		{
			name:             "OCM type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeOCM),
			expected:         KubeconfigSecretKeyDefault,
		},
		{
			name:             "VCluster type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeVCluster),
			expected:         KubeconfigSecretKeyVCluster,
		},
		{
			name:             "K3s type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK3s),
			expected:         KubeconfigSecretKeyVCluster,
		},
		{
			name:             "unknown type",
			controlPlaneType: "unknown",
			expected:         KubeconfigSecretKeyDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKubeconfSecretKeyNameByControlPlaneType(tt.controlPlaneType)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetAPIServerDeploymentNameByControlPlaneType(t *testing.T) {
	tests := []struct {
		name             string
		controlPlaneType string
		expected         string
	}{
		{
			name:             "K8S type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK8S),
			expected:         APIServerDeploymentName,
		},
		{
			name:             "OCM type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeOCM),
			expected:         OCMServerDeploymentName,
		},
		{
			name:             "VCluster type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeVCluster),
			expected:         VClusterServerDeploymentName,
		},
		{
			name:             "K3s type",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK3s),
			expected:         K3sServerDeploymentName,
		},
		{
			name:             "unknown type",
			controlPlaneType: "unknown",
			expected:         APIServerDeploymentName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAPIServerDeploymentNameByControlPlaneType(tt.controlPlaneType)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsInCluster(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "in cluster",
			envValue: "kubernetes.default.svc",
			expected: true,
		},
		{
			name:     "not in cluster",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env value
			originalValue := os.Getenv("KUBERNETES_SERVICE_HOST")
			defer func() {
				if err := os.Setenv("KUBERNETES_SERVICE_HOST", originalValue); err != nil {
					t.Errorf("failed to restore KUBERNETES_SERVICE_HOST: %v", err)
				}
			}()

			// Set test env value
			if tt.envValue != "" {
				if err := os.Setenv("KUBERNETES_SERVICE_HOST", tt.envValue); err != nil {
					t.Fatalf("failed to set KUBERNETES_SERVICE_HOST: %v", err)
				}
			} else {
				if err := os.Unsetenv("KUBERNETES_SERVICE_HOST"); err != nil {
					t.Fatalf("failed to unset KUBERNETES_SERVICE_HOST: %v", err)
				}
			}

			result := IsInCluster()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestZeroFields(t *testing.T) {
	// Create a test ConfigMap with metadata fields
	original := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-cm",
			Namespace:       "test-ns",
			ResourceVersion: "12345",
			Generation:      10,
			UID:             types.UID("test-uid"),
			CreationTimestamp: metav1.Time{
				Time: metav1.Now().Time,
			},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{
					Manager: "test-manager",
				},
			},
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	result := ZeroFields(original)
	resultCM := result.(*corev1.ConfigMap)

	// Verify zeroed fields
	if resultCM.ResourceVersion != "" {
		t.Errorf("expected empty ResourceVersion, got %q", resultCM.ResourceVersion)
	}
	if resultCM.Generation != 0 {
		t.Errorf("expected Generation 0, got %d", resultCM.Generation)
	}
	if resultCM.UID != "" {
		t.Errorf("expected empty UID, got %q", resultCM.UID)
	}
	if !resultCM.CreationTimestamp.IsZero() {
		t.Error("expected zero CreationTimestamp")
	}
	if resultCM.ManagedFields != nil {
		t.Error("expected nil ManagedFields")
	}

	// Verify non-zeroed fields (preserved)
	if resultCM.Name != original.Name {
		t.Errorf("expected Name %q, got %q", original.Name, resultCM.Name)
	}
	if resultCM.Namespace != original.Namespace {
		t.Errorf("expected Namespace %q, got %q", original.Namespace, resultCM.Namespace)
	}
	if resultCM.Data["key"] != "value" {
		t.Error("expected Data to be preserved")
	}

	// Verify original is unchanged
	if original.ResourceVersion == "" {
		t.Error("original ResourceVersion should not be modified")
	}
}

func TestDefaultString(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue string
		expected     string
	}{
		{
			name:         "value provided",
			value:        "custom-value",
			defaultValue: "default-value",
			expected:     "custom-value",
		},
		{
			name:         "empty value",
			value:        "",
			defaultValue: "default-value",
			expected:     "default-value",
		},
		{
			name:         "empty default",
			value:        "",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultString(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateBootstrapSecretName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		expected string
	}{
		{
			name:     "standard control plane name",
			cpName:   "my-cp",
			expected: "my-cp-bootstrap",
		},
		{
			name:     "empty control plane name",
			cpName:   "",
			expected: "-bootstrap",
		},
		{
			name:     "control plane with hyphens",
			cpName:   "test-cp-123",
			expected: "test-cp-123-bootstrap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateBootstrapSecretName(tt.cpName)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestGetKubernetesClusterVersionInfo tests the GetKubernetesClusterVersionInfo function
func TestGetKubernetesClusterVersionInfo(t *testing.T) {
	// Test with invalid kubeconfig path - should return error
	t.Run("invalid kubeconfig", func(t *testing.T) {
		_, err := GetKubernetesClusterVersionInfo("/tmp/nonexistent-kubeconfig-xyz123")
		if err == nil {
			t.Error("expected error for invalid kubeconfig but got none")
		}
	})
}
