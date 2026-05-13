package certs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// Helper function to create a test CA
func createTestCA(t *testing.T) (*rsa.PrivateKey, x509.Certificate, []byte) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "kubernetes",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caPEMCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	})

	// Add RawSubjectPublicKeyInfo for the template
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&caKey.PublicKey)
	require.NoError(t, err)
	caTemplate.RawSubjectPublicKeyInfo = pubKeyBytes

	return caKey, caTemplate, caPEMCert
}

// Helper function to create test Certs object
func createTestCerts(t *testing.T) *Certs {
	caKey, caTemplate, caPEMCert := createTestCA(t)
	return &Certs{
		caKey:      caKey,
		caTemplate: caTemplate,
		caPEMCert:  caPEMCert,
	}
}

func TestGenerateClusterName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		expected string
	}{
		{
			name:     "simple name",
			cpName:   "test-cp",
			expected: "test-cp-cluster",
		},
		{
			name:     "name with numbers",
			cpName:   "cp-123",
			expected: "cp-123-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateClusterName(tt.cpName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateAuthInfoAdminName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		expected string
	}{
		{
			name:     "simple name",
			cpName:   "test-cp",
			expected: "test-cp-admin",
		},
		{
			name:     "name with hyphens",
			cpName:   "my-control-plane",
			expected: "my-control-plane-admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateAuthInfoAdminName(tt.cpName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateContextName(t *testing.T) {
	tests := []struct {
		name     string
		cpName   string
		expected string
	}{
		{
			name:     "context name matches cp name",
			cpName:   "test-cp",
			expected: "test-cp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateContextName(tt.cpName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigGen_generateServerEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		config      *ConfigGen
		expected    string
		description string
	}{
		{
			name: "ControllerManager target",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				Target:      ControllerManager,
			},
			expected:    "https://test-cp.test-ns.svc.cluster.local",
			description: "Should use in-cluster DNS for controller manager",
		},
		{
			name: "AdminInCluster target",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				Target:      AdminInCluster,
			},
			expected:    "https://test-cp.test-ns.svc.cluster.local",
			description: "Should use in-cluster DNS for admin in cluster",
		},
		{
			name: "Admin target with ExtraDNS",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				CpExtraDNS:  "api.example.com",
				Target:      Admin,
			},
			expected:    "https://api.example.com",
			description: "Should use ExtraDNS when provided",
		},
		{
			name: "Admin target without ExtraDNS",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				CpDomain:    "example.com",
				CpPort:      6443,
				Target:      Admin,
			},
			expected:    "https://test-cp.example.com:6443",
			description: "Should use domain and port when ExtraDNS not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.generateServerEndpoint()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestConfigGen_generateConfigCerts(t *testing.T) {
	caKey, caTemplate, caPEMCert := createTestCA(t)

	tests := []struct {
		name           string
		target         ConfigTarget
		expectedCN     string
		expectedOrg    []string
		expectedAuth   string
		expectedSecret string
		expectError    bool
	}{
		{
			name:           "Admin target",
			target:         Admin,
			expectedCN:     AdminCN,
			expectedOrg:    []string{Organization},
			expectedAuth:   "test-cp-admin",
			expectedSecret: util.AdminConfSecret,
			expectError:    false,
		},
		{
			name:           "AdminInCluster target",
			target:         AdminInCluster,
			expectedCN:     AdminCN,
			expectedOrg:    []string{Organization},
			expectedAuth:   "test-cp-admin",
			expectedSecret: util.AdminConfSecret,
			expectError:    false,
		},
		{
			name:           "ControllerManager target",
			target:         ControllerManager,
			expectedCN:     ContrCMCN,
			expectedOrg:    nil,
			expectedAuth:   ContrCMCN,
			expectedSecret: CMConfSecret,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				Target:      tt.target,
				caKey:       caKey,
				caTemplate:  caTemplate,
				caPEMCert:   caPEMCert,
			}

			err := conf.generateConfigCerts()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, conf.key)
			assert.NotNil(t, conf.cert)
			assert.Equal(t, tt.expectedAuth, conf.authInfo)
			assert.Equal(t, tt.expectedSecret, conf.secretName)

			// Verify certificate can be decoded
			block, _ := pem.Decode(conf.cert)
			require.NotNil(t, block)
			cert, err := x509.ParseCertificate(block.Bytes)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCN, cert.Subject.CommonName)
			if tt.expectedOrg != nil {
				assert.Equal(t, tt.expectedOrg, cert.Subject.Organization)
			}

			// Verify key can be decoded
			keyBlock, _ := pem.Decode(conf.key)
			require.NotNil(t, keyBlock)
			_, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
			require.NoError(t, err)
		})
	}
}

func TestConfigGen_generateConfig(t *testing.T) {
	caKey, caTemplate, caPEMCert := createTestCA(t)

	conf := &ConfigGen{
		CpName:      "test-cp",
		CpNamespace: "test-ns",
		CpDomain:    "example.com",
		CpPort:      6443,
		Target:      Admin,
		caKey:       caKey,
		caTemplate:  caTemplate,
		caPEMCert:   caPEMCert,
	}

	err := conf.generateConfigCerts()
	require.NoError(t, err)

	config := conf.generateConfig()

	assert.NotNil(t, config)
	assert.Equal(t, GenerateContextName(conf.CpName), config.CurrentContext)

	// Verify cluster
	cluster, exists := config.Clusters[GenerateClusterName(conf.CpName)]
	require.True(t, exists)
	assert.NotEmpty(t, cluster.Server)
	assert.Equal(t, caPEMCert, cluster.CertificateAuthorityData)

	// Verify auth info
	authInfo, exists := config.AuthInfos[conf.authInfo]
	require.True(t, exists)
	assert.Equal(t, conf.cert, authInfo.ClientCertificateData)
	assert.Equal(t, conf.key, authInfo.ClientKeyData)

	// Verify context
	ctx, exists := config.Contexts[GenerateContextName(conf.CpName)]
	require.True(t, exists)
	assert.Equal(t, GenerateClusterName(conf.CpName), ctx.Cluster)
	assert.Equal(t, DefaultNamespace, ctx.Namespace)
	assert.Equal(t, conf.authInfo, ctx.AuthInfo)
}

func TestGenerateKubeconfigBytes(t *testing.T) {
	caKey, caTemplate, caPEMCert := createTestCA(t)

	conf := &ConfigGen{
		CpName:      "test-cp",
		CpNamespace: "test-ns",
		CpDomain:    "example.com",
		CpPort:      6443,
		Target:      Admin,
		caKey:       caKey,
		caTemplate:  caTemplate,
		caPEMCert:   caPEMCert,
	}

	kubeconfigBytes, err := GenerateKubeconfigBytes(conf)
	require.NoError(t, err)
	assert.NotEmpty(t, kubeconfigBytes)

	// Verify it can be loaded as a valid kubeconfig
	config, err := clientcmd.Load(kubeconfigBytes)
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, GenerateContextName(conf.CpName), config.CurrentContext)
}

func TestGenerateKubeConfigSecret(t *testing.T) {
	certs := createTestCerts(t)
	ctx := context.Background()

	tests := []struct {
		name              string
		target            ConfigTarget
		expectInCluster   bool
		expectedSecretKey string
	}{
		{
			name:              "Admin config generates both kubeconfigs",
			target:            Admin,
			expectInCluster:   true,
			expectedSecretKey: util.AdminConfSecret,
		},
		{
			name:              "ControllerManager config generates one kubeconfig",
			target:            ControllerManager,
			expectInCluster:   false,
			expectedSecretKey: CMConfSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				CpDomain:    "example.com",
				CpPort:      6443,
				Target:      tt.target,
			}

			secret, err := GenerateKubeConfigSecret(ctx, certs, conf)
			require.NoError(t, err)
			require.NotNil(t, secret)

			assert.Equal(t, conf.CpNamespace, secret.Namespace)
			assert.Equal(t, v1.SecretTypeOpaque, secret.Type)

			// Verify default kubeconfig exists
			defaultKubeconfig, exists := secret.Data[util.KubeconfigSecretKeyDefault]
			require.True(t, exists)
			assert.NotEmpty(t, defaultKubeconfig)

			// Verify in-cluster kubeconfig exists only for Admin target
			inClusterKubeconfig, exists := secret.Data[util.KubeconfigSecretKeyInCluster]
			if tt.expectInCluster {
				require.True(t, exists)
				assert.NotEmpty(t, inClusterKubeconfig)

				// Verify both kubeconfigs are valid
				_, err = clientcmd.Load(defaultKubeconfig)
				require.NoError(t, err)

				inClusterConfig, err := clientcmd.Load(inClusterKubeconfig)
				require.NoError(t, err)

				// Verify in-cluster config uses in-cluster endpoint
				clusterName := GenerateClusterName(conf.CpName)
				cluster := inClusterConfig.Clusters[clusterName]
				require.NotNil(t, cluster)
				assert.Contains(t, cluster.Server, "svc.cluster.local")
			} else {
				assert.False(t, exists)
			}
		})
	}
}

func TestConfigGen_genSecretManifest(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		config          *ConfigGen
		kconf           []byte
		kconfInCluster  []byte
		expectedDataLen int
	}{
		{
			name: "Secret with only default kubeconfig",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				secretName:  "test-secret",
			},
			kconf:           []byte("default-kubeconfig"),
			kconfInCluster:  nil,
			expectedDataLen: 1,
		},
		{
			name: "Secret with both kubeconfigs",
			config: &ConfigGen{
				CpName:      "test-cp",
				CpNamespace: "test-ns",
				secretName:  "test-secret",
			},
			kconf:           []byte("default-kubeconfig"),
			kconfInCluster:  []byte("incluster-kubeconfig"),
			expectedDataLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := tt.config.genSecretManifest(ctx, tt.kconf, tt.kconfInCluster)

			assert.NotNil(t, secret)
			assert.Equal(t, tt.config.secretName, secret.Name)
			assert.Equal(t, tt.config.CpNamespace, secret.Namespace)
			assert.Equal(t, v1.SecretTypeOpaque, secret.Type)
			assert.Len(t, secret.Data, tt.expectedDataLen)

			assert.Equal(t, tt.kconf, secret.Data[util.KubeconfigSecretKeyDefault])

			if tt.kconfInCluster != nil {
				assert.Equal(t, tt.kconfInCluster, secret.Data[util.KubeconfigSecretKeyInCluster])
			}
		})
	}
}
