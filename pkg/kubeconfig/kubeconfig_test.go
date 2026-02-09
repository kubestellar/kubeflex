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

package kubeconfig

import (
	"testing"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/certs"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestAdjustConfigKeys(t *testing.T) {
	tests := []struct {
		name             string
		cpName           string
		controlPlaneType string
		setupConfig      func() *clientcmdapi.Config
		validate         func(t *testing.T, kconf *clientcmdapi.Config, cpName string)
	}{
		{
			name:             "OCM control plane type",
			cpName:           "test-cp",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeOCM),
			setupConfig: func() *clientcmdapi.Config {
				return &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						ControlPlaneTypeOCMDefault: {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"user": {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						ControlPlaneTypeOCMDefault: {
							Cluster:  ControlPlaneTypeOCMDefault,
							AuthInfo: "user",
						},
					},
				}
			},
			validate: func(t *testing.T, kconf *clientcmdapi.Config, cpName string) {
				expectedCluster := certs.GenerateClusterName(cpName)
				expectedAuthInfo := certs.GenerateAuthInfoAdminName(cpName)
				expectedContext := certs.GenerateContextName(cpName)

				if _, ok := kconf.Clusters[expectedCluster]; !ok {
					t.Errorf("expected cluster %s not found", expectedCluster)
				}
				if _, ok := kconf.AuthInfos[expectedAuthInfo]; !ok {
					t.Errorf("expected authInfo %s not found", expectedAuthInfo)
				}
				if _, ok := kconf.Contexts[expectedContext]; !ok {
					t.Errorf("expected context %s not found", expectedContext)
				}
				if kconf.CurrentContext != expectedContext {
					t.Errorf("expected current context %s, got %s", expectedContext, kconf.CurrentContext)
				}
			},
		},
		{
			name:             "VCluster control plane type",
			cpName:           "vcluster-cp",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeVCluster),
			setupConfig: func() *clientcmdapi.Config {
				return &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						ControlPlaneTypeVClusterDefault: {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						ControlPlaneTypeVClusterDefault: {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						ControlPlaneTypeVClusterDefault: {
							Cluster:  ControlPlaneTypeVClusterDefault,
							AuthInfo: ControlPlaneTypeVClusterDefault,
						},
					},
				}
			},
			validate: func(t *testing.T, kconf *clientcmdapi.Config, cpName string) {
				expectedContext := certs.GenerateContextName(cpName)
				if kconf.CurrentContext != expectedContext {
					t.Errorf("expected current context %s, got %s", expectedContext, kconf.CurrentContext)
				}
			},
		},
		{
			name:             "K3s control plane type",
			cpName:           "k3s-cp",
			controlPlaneType: string(tenancyv1alpha1.ControlPlaneTypeK3s),
			setupConfig: func() *clientcmdapi.Config {
				return &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						ControlPlaneTypeK3sDefault: {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						ControlPlaneTypeK3sDefault: {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						ControlPlaneTypeK3sDefault: {
							Cluster:  ControlPlaneTypeK3sDefault,
							AuthInfo: ControlPlaneTypeK3sDefault,
						},
					},
				}
			},
			validate: func(t *testing.T, kconf *clientcmdapi.Config, cpName string) {
				expectedContext := certs.GenerateContextName(cpName)
				if kconf.CurrentContext != expectedContext {
					t.Errorf("expected current context %s, got %s", expectedContext, kconf.CurrentContext)
				}
			},
		},
		{
			name:             "Unknown control plane type - no changes",
			cpName:           "unknown-cp",
			controlPlaneType: "unknown-type",
			setupConfig: func() *clientcmdapi.Config {
				return &clientcmdapi.Config{
					Clusters:       map[string]*clientcmdapi.Cluster{},
					AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
					Contexts:       map[string]*clientcmdapi.Context{},
					CurrentContext: "original-context",
				}
			},
			validate: func(t *testing.T, kconf *clientcmdapi.Config, cpName string) {
				if kconf.CurrentContext != "original-context" {
					t.Errorf("expected current context to remain 'original-context', got %s", kconf.CurrentContext)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kconf := tt.setupConfig()
			adjustConfigKeys(kconf, tt.cpName, tt.controlPlaneType)
			tt.validate(t, kconf, tt.cpName)
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name        string
		base        *clientcmdapi.Config
		target      *clientcmdapi.Config
		expectError bool
		validate    func(t *testing.T, result *clientcmdapi.Config)
	}{
		{
			name: "successful merge with hosting cluster context set",
			base: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"base-cluster": {Server: "https://base:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"base-user": {Token: "base-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"base-context": {
						Cluster:  "base-cluster",
						AuthInfo: "base-user",
					},
				},
				CurrentContext: "base-context",
				Extensions: map[string]runtime.Object{
					ExtensionKubeflexKey: &runtime.Unknown{
						Raw: []byte(`{"hostingClusterContextName":"base-context"}`),
					},
				},
			},
			target: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"target-cluster": {Server: "https://target:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"target-user": {Token: "target-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"target-context": {
						Cluster:  "target-cluster",
						AuthInfo: "target-user",
					},
				},
				CurrentContext: "target-context",
			},
			expectError: false,
			validate: func(t *testing.T, result *clientcmdapi.Config) {
				if len(result.Clusters) != 2 {
					t.Errorf("expected 2 clusters, got %d", len(result.Clusters))
				}
				if len(result.AuthInfos) != 2 {
					t.Errorf("expected 2 authInfos, got %d", len(result.AuthInfos))
				}
				if len(result.Contexts) != 2 {
					t.Errorf("expected 2 contexts, got %d", len(result.Contexts))
				}
				if result.CurrentContext != "target-context" {
					t.Errorf("expected current context 'target-context', got %s", result.CurrentContext)
				}
			},
		},
		{
			name: "merge without hosting cluster context - should set it",
			base: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"base-cluster": {Server: "https://base:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"base-user": {Token: "base-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"base-context": {
						Cluster:  "base-cluster",
						AuthInfo: "base-user",
					},
				},
				CurrentContext: "base-context",
			},
			target: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"target-cluster": {Server: "https://target:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"target-user": {Token: "target-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"target-context": {
						Cluster:  "target-cluster",
						AuthInfo: "target-user",
					},
				},
				CurrentContext: "target-context",
			},
			expectError: false,
			validate: func(t *testing.T, result *clientcmdapi.Config) {
				if result.Extensions == nil {
					t.Error("expected extensions to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := merge(tt.base, tt.target)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, tt.base)
			}
		})
	}
}

func TestAssignControlPlaneToContext(t *testing.T) {
	tests := []struct {
		name        string
		kconf       *clientcmdapi.Config
		cpName      string
		ctxName     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully assign control plane to existing context",
			kconf: &clientcmdapi.Config{
				Contexts: map[string]*clientcmdapi.Context{
					"test-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
					},
				},
			},
			cpName:      "test-cp",
			ctxName:     "test-context",
			expectError: false,
		},
		{
			name: "error when context does not exist",
			kconf: &clientcmdapi.Config{
				Contexts: map[string]*clientcmdapi.Context{},
			},
			cpName:      "test-cp",
			ctxName:     "non-existent-context",
			expectError: true,
			errorMsg:    "error context non-existent-context does not exist in config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AssignControlPlaneToContext(tt.kconf, tt.cpName, tt.ctxName)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err != nil && tt.errorMsg != "" && err.Error() != tt.errorMsg {
				t.Errorf("expected error message: %s, got: %s", tt.errorMsg, err.Error())
			}
		})
	}
}

func TestDeleteAll(t *testing.T) {
	tests := []struct {
		name        string
		kconf       *clientcmdapi.Config
		cpName      string
		expectError bool
		validate    func(t *testing.T, kconf *clientcmdapi.Config)
	}{
		{
			name: "successfully delete all control plane resources",
			kconf: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"test-cp-cluster": {Server: "https://localhost:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"test-cp-admin": {Token: "test-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"test-cp": {
						Cluster:  "test-cp-cluster",
						AuthInfo: "test-cp-admin",
					},
				},
			},
			cpName:      "test-cp",
			expectError: false,
			validate: func(t *testing.T, kconf *clientcmdapi.Config) {
				if len(kconf.Clusters) != 0 {
					t.Errorf("expected 0 clusters, got %d", len(kconf.Clusters))
				}
				if len(kconf.AuthInfos) != 0 {
					t.Errorf("expected 0 authInfos, got %d", len(kconf.AuthInfos))
				}
				if len(kconf.Contexts) != 0 {
					t.Errorf("expected 0 contexts, got %d", len(kconf.Contexts))
				}
			},
		},
		{
			name: "error when context not found",
			kconf: &clientcmdapi.Config{
				Clusters:  map[string]*clientcmdapi.Cluster{},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{},
				Contexts:  map[string]*clientcmdapi.Context{},
			},
			cpName:      "non-existent-cp",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteAll(tt.kconf, tt.cpName)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, tt.kconf)
			}
		})
	}
}

func TestRenameKey(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		oldKey   string
		newKey   string
		validate func(t *testing.T, result interface{})
	}{
		{
			name: "rename cluster key",
			input: map[string]*clientcmdapi.Cluster{
				"old-cluster": {Server: "https://localhost:6443"},
			},
			oldKey: "old-cluster",
			newKey: "new-cluster",
			validate: func(t *testing.T, result interface{}) {
				clusters := result.(map[string]*clientcmdapi.Cluster)
				if _, ok := clusters["old-cluster"]; ok {
					t.Error("old key should not exist")
				}
				if _, ok := clusters["new-cluster"]; !ok {
					t.Error("new key should exist")
				}
			},
		},
		{
			name: "rename authInfo key",
			input: map[string]*clientcmdapi.AuthInfo{
				"old-user": {Token: "test-token"},
			},
			oldKey: "old-user",
			newKey: "new-user",
			validate: func(t *testing.T, result interface{}) {
				authInfos := result.(map[string]*clientcmdapi.AuthInfo)
				if _, ok := authInfos["old-user"]; ok {
					t.Error("old key should not exist")
				}
				if _, ok := authInfos["new-user"]; !ok {
					t.Error("new key should exist")
				}
			},
		},
		{
			name: "rename context key",
			input: map[string]*clientcmdapi.Context{
				"old-context": {Cluster: "test-cluster"},
			},
			oldKey: "old-context",
			newKey: "new-context",
			validate: func(t *testing.T, result interface{}) {
				contexts := result.(map[string]*clientcmdapi.Context)
				if _, ok := contexts["old-context"]; ok {
					t.Error("old key should not exist")
				}
				if _, ok := contexts["new-context"]; !ok {
					t.Error("new key should exist")
				}
			},
		},
		{
			name: "rename non-existent key - no change",
			input: map[string]*clientcmdapi.Cluster{
				"existing-cluster": {Server: "https://localhost:6443"},
			},
			oldKey: "non-existent",
			newKey: "new-cluster",
			validate: func(t *testing.T, result interface{}) {
				clusters := result.(map[string]*clientcmdapi.Cluster)
				if _, ok := clusters["existing-cluster"]; !ok {
					t.Error("existing cluster should still exist")
				}
				if _, ok := clusters["new-cluster"]; ok {
					t.Error("new cluster should not be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RenameKey(tt.input, tt.oldKey, tt.newKey)
			tt.validate(t, tt.input)
		})
	}
}

func TestSwitchContext(t *testing.T) {
	tests := []struct {
		name        string
		kconf       *clientcmdapi.Config
		cpName      string
		expectError bool
		validate    func(t *testing.T, kconf *clientcmdapi.Config)
	}{
		{
			name: "successfully switch context",
			kconf: &clientcmdapi.Config{
				Contexts: map[string]*clientcmdapi.Context{
					"test-cp": {
						Cluster:  "test-cp-cluster",
						AuthInfo: "test-cp-admin",
					},
				},
				CurrentContext: "old-context",
			},
			cpName:      "test-cp",
			expectError: false,
			validate: func(t *testing.T, kconf *clientcmdapi.Config) {
				if kconf.CurrentContext != "test-cp" {
					t.Errorf("expected current context 'test-cp', got %s", kconf.CurrentContext)
				}
			},
		},
		{
			name: "error when context not found",
			kconf: &clientcmdapi.Config{
				Contexts:       map[string]*clientcmdapi.Context{},
				CurrentContext: "old-context",
			},
			cpName:      "non-existent-cp",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SwitchContext(tt.kconf, tt.cpName)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, tt.kconf)
			}
		})
	}
}

func TestSetHostingClusterContext(t *testing.T) {
	customCtxName := "custom-context"

	tests := []struct {
		name        string
		kconf       *clientcmdapi.Config
		ctxName     *string
		expectError bool
		validate    func(t *testing.T, kconf *clientcmdapi.Config)
	}{
		{
			name: "set to current context when ctxName is nil",
			kconf: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"test-cluster": {Server: "https://localhost:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"test-user": {Token: "test-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"current-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
					},
				},
				CurrentContext: "current-context",
			},
			ctxName:     nil,
			expectError: false,
			validate: func(t *testing.T, kconf *clientcmdapi.Config) {
				hostingCtx, err := GetHostingClusterContext(kconf)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if hostingCtx != "current-context" {
					t.Errorf("expected hosting context 'current-context', got %s", hostingCtx)
				}
			},
		},
		{
			name: "set to specified context when ctxName provided",
			kconf: &clientcmdapi.Config{
				Clusters: map[string]*clientcmdapi.Cluster{
					"test-cluster": {Server: "https://localhost:6443"},
				},
				AuthInfos: map[string]*clientcmdapi.AuthInfo{
					"test-user": {Token: "test-token"},
				},
				Contexts: map[string]*clientcmdapi.Context{
					"custom-context": {
						Cluster:  "test-cluster",
						AuthInfo: "test-user",
					},
				},
				CurrentContext: "other-context",
			},
			ctxName:     &customCtxName,
			expectError: false,
			validate: func(t *testing.T, kconf *clientcmdapi.Config) {
				hostingCtx, err := GetHostingClusterContext(kconf)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if hostingCtx != "custom-context" {
					t.Errorf("expected hosting context 'custom-context', got %s", hostingCtx)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetHostingClusterContext(tt.kconf, tt.ctxName)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, tt.kconf)
			}
		})
	}
}

func TestGetHostingClusterContext(t *testing.T) {
	tests := []struct {
		name          string
		kconf         *clientcmdapi.Config
		expectError   bool
		expectedCtx   string
		errorContains string
	}{
		{
			name: "successfully get hosting cluster context",
			kconf: func() *clientcmdapi.Config {
				cfg := &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						"test-cluster": {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"test-user": {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						"hosting-context": {
							Cluster:  "test-cluster",
							AuthInfo: "test-user",
						},
					},
					CurrentContext: "hosting-context",
				}
				// Properly set hosting cluster context using the function
				if err := SetHostingClusterContext(cfg, nil); err != nil {
					t.Fatalf("failed to set hosting cluster context: %v", err)
				}
				return cfg
			}(),
			expectError: false,
			expectedCtx: "hosting-context",
		},
		{
			name: "error when hosting cluster context not set",
			kconf: &clientcmdapi.Config{
				Contexts: map[string]*clientcmdapi.Context{},
			},
			expectError:   true,
			errorContains: "hosting cluster context data not set",
		},
		{
			name: "error when referenced context does not exist",
			kconf: &clientcmdapi.Config{
				Contexts: map[string]*clientcmdapi.Context{},
				Extensions: map[string]runtime.Object{
					ExtensionKubeflexKey: &runtime.Unknown{
						Raw: []byte(`{"hostingClusterContextName":"non-existent"}`),
					},
				},
			},
			expectError:   true,
			errorContains: "non-existing context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := GetHostingClusterContext(tt.kconf)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && ctx != tt.expectedCtx {
				t.Errorf("expected context %s, got %s", tt.expectedCtx, ctx)
			}
		})
	}
}

func TestIsHostingClusterContextSet(t *testing.T) {
	tests := []struct {
		name     string
		kconf    *clientcmdapi.Config
		expected bool
	}{
		{
			name: "hosting cluster context is set",
			kconf: func() *clientcmdapi.Config {
				cfg := &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						"test-cluster": {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"test-user": {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						"hosting-context": {
							Cluster:  "test-cluster",
							AuthInfo: "test-user",
						},
					},
					CurrentContext: "hosting-context",
				}
				// Properly set the hosting cluster context
				if err := SetHostingClusterContext(cfg, nil); err != nil {
					t.Fatalf("failed to set hosting cluster context: %v", err)
				}
				return cfg
			}(),
			expected: true,
		},
		{
			name: "hosting cluster context is not set",
			kconf: &clientcmdapi.Config{
				Extensions: map[string]runtime.Object{},
			},
			expected: false,
		},
		{
			name: "invalid extensions",
			kconf: &clientcmdapi.Config{
				Extensions: map[string]runtime.Object{
					ExtensionKubeflexKey: &runtime.Unknown{
						Raw: []byte(`invalid json`),
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHostingClusterContextSet(tt.kconf)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSwitchToHostingClusterContext(t *testing.T) {
	tests := []struct {
		name        string
		kconf       *clientcmdapi.Config
		expectError bool
		validate    func(t *testing.T, kconf *clientcmdapi.Config)
	}{
		{
			name: "successfully switch to hosting cluster context",
			kconf: func() *clientcmdapi.Config {
				cfg := &clientcmdapi.Config{
					Clusters: map[string]*clientcmdapi.Cluster{
						"test-cluster": {Server: "https://localhost:6443"},
					},
					AuthInfos: map[string]*clientcmdapi.AuthInfo{
						"test-user": {Token: "test-token"},
					},
					Contexts: map[string]*clientcmdapi.Context{
						"hosting-context": {
							Cluster:  "test-cluster",
							AuthInfo: "test-user",
						},
						"other-context": {
							Cluster:  "test-cluster",
							AuthInfo: "test-user",
						},
					},
					CurrentContext: "other-context",
				}
				// Set hosting cluster context first
				hostingCtx := "hosting-context"
				if err := SetHostingClusterContext(cfg, &hostingCtx); err != nil {
					t.Fatalf("failed to set hosting cluster context: %v", err)
				}
				// Then set current context to something else to test the switch
				cfg.CurrentContext = "other-context"
				return cfg
			}(),
			expectError: false,
			validate: func(t *testing.T, kconf *clientcmdapi.Config) {
				if kconf.CurrentContext != "hosting-context" {
					t.Errorf("expected current context 'hosting-context', got %s", kconf.CurrentContext)
				}
			},
		},
		{
			name: "error when hosting cluster context not set",
			kconf: &clientcmdapi.Config{
				Contexts:       map[string]*clientcmdapi.Context{},
				CurrentContext: "other-context",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SwitchToHostingClusterContext(tt.kconf)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}
			if err == nil && tt.validate != nil {
				tt.validate(t, tt.kconf)
			}
		})
	}
}
