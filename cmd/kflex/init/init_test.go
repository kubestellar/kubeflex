/*
Copyright 2025 The KubeStellar Authors.

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

package init

import "testing"

func TestDefaultKindClusterNameConstant(t *testing.T) {
	// Test that the DefaultKindClusterName constant is properly defined
	if DefaultKindClusterName == "" {
		t.Error("DefaultKindClusterName constant should not be empty")
	}
}

func TestDefaultKindClusterNameValue(t *testing.T) {
	// Test that the DefaultKindClusterName constant has the expected value
	expected := "kind-kubeflex"
	if DefaultKindClusterName != expected {
		t.Errorf("DefaultKindClusterName constant should be '%s', got '%s'", expected, DefaultKindClusterName)
	}
}

func TestDefaultKindClusterNameConsistency(t *testing.T) {
	// Test that the DefaultKindClusterName constant value is consistent
	// This ensures that if the constant value is changed, tests will catch it
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "DefaultKindClusterName matches expected format",
			constant: DefaultKindClusterName,
			expected: "kind-kubeflex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("constant %s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestDefaultKindClusterNameUsage(t *testing.T) {
	// Test that the DefaultKindClusterName constant follows the expected naming convention
	// This ensures the constant follows the "kind-" prefix pattern used by kind clusters
	if len(DefaultKindClusterName) <= 5 {
		t.Error("DefaultKindClusterName should be longer than 5 characters")
	}

	if DefaultKindClusterName[:5] != "kind-" {
		t.Error("DefaultKindClusterName should start with 'kind-' prefix")
	}
}
