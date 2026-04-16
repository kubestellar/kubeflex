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

package client

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// createTestKubeconfig creates a temporary kubeconfig file for testing
func createTestKubeconfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	config := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {
				Server:                "https://localhost:6443",
				InsecureSkipTLSVerify: true,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:  "test-cluster",
				AuthInfo: "test-user",
			},
		},
		CurrentContext: "test-context",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {
				Token: "test-token",
			},
		},
	}

	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	return kubeconfigPath
}

// createInvalidKubeconfig creates an invalid kubeconfig file
func createInvalidKubeconfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	if err := os.WriteFile(kubeconfigPath, []byte("invalid yaml content: [[["), 0600); err != nil {
		t.Fatalf("Failed to write invalid kubeconfig: %v", err)
	}

	return kubeconfigPath
}
func TestGetClientSet(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		setupFunc   func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid kubeconfig",
			setupFunc: func() string {
				return createTestKubeconfig(t)
			},
			wantErr: false,
		},
		{
			name: "invalid kubeconfig",
			setupFunc: func() string {
				return createInvalidKubeconfig(t)
			},
			wantErr:     true,
			errContains: "error building kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfig := tt.setupFunc()

			clientset, err := GetClientSet(kubeconfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetClientSet() expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetClientSet() error = %v, should contain %v", err, tt.errContains)
				}
				if clientset != nil {
					t.Errorf("GetClientSet() expected nil clientset on error")
				}
			} else {
				if err != nil {
					t.Errorf("GetClientSet() unexpected error: %v", err)
				}
				if clientset == nil {
					t.Errorf("GetClientSet() returned nil clientset")
				}
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid kubeconfig",
			setupFunc: func() string {
				return createTestKubeconfig(t)
			},
			wantErr: false,
		},
		{
			name: "invalid kubeconfig",
			setupFunc: func() string {
				return createInvalidKubeconfig(t)
			},
			wantErr:     true,
			errContains: "error building kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfig := tt.setupFunc()

			client, err := GetClient(kubeconfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetClient() expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetClient() error = %v, should contain %v", err, tt.errContains)
				}
				if client != nil {
					t.Errorf("GetClient() expected nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("GetClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("GetClient() returned nil client")
				}
			}
		})
	}
}

func TestGetOpendShiftSecClient(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid kubeconfig",
			setupFunc: func() string {
				return createTestKubeconfig(t)
			},
			wantErr: false,
		},
		{
			name: "invalid kubeconfig",
			setupFunc: func() string {
				return createInvalidKubeconfig(t)
			},
			wantErr:     true,
			errContains: "error building kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeconfig := tt.setupFunc()

			client, err := GetOpendShiftSecClient(kubeconfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetOpendShiftSecClient() expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetOpendShiftSecClient() error = %v, should contain %v", err, tt.errContains)
				}
				if client != nil {
					t.Errorf("GetOpendShiftSecClient() expected nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("GetOpendShiftSecClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("GetOpendShiftSecClient() returned nil client")
				}
			}
		})
	}
}

func TestGetConfigWithHomeDirFallback(t *testing.T) {
	// Save and restore original HOME/USERPROFILE
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Setenv("USERPROFILE", originalUserProfile)
	}()

	// Unset KUBECONFIG to force home dir lookup
	originalKubeconfig := os.Getenv("KUBECONFIG")
	os.Unsetenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalKubeconfig)

	// Create a temporary home directory with .kube/config
	tmpHome := t.TempDir()
	kubeDir := filepath.Join(tmpHome, ".kube")
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		t.Fatalf("Failed to create .kube directory: %v", err)
	}

	// Create a valid kubeconfig in the temp home
	config := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test-cluster": {
				Server:                "https://localhost:6443",
				InsecureSkipTLSVerify: true,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test-context": {
				Cluster:  "test-cluster",
				AuthInfo: "test-user",
			},
		},
		CurrentContext: "test-context",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test-user": {
				Token: "test-token",
			},
		},
	}

	kubeconfigPath := filepath.Join(kubeDir, "config")
	if err := clientcmd.WriteToFile(config, kubeconfigPath); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Set HOME to temp directory
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome) // For Windows compatibility

	// Test with empty kubeconfig parameter
	cfg, err := getConfig("")
	if err != nil {
		t.Errorf("getConfig() with home dir fallback failed: %v", err)
	}
	if cfg == nil {
		t.Errorf("getConfig() returned nil config")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
