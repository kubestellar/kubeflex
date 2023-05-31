package certs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"mcc.ibm.org/kubeflex/pkg/util"
)

type ConfigTarget int

const (
	Admin ConfigTarget = iota
	ControllerManager
	DefaultNamespace = "default"
	AdminCN          = "kubernetes-admin"
	Organization     = "system:masters"
	ContrCMCN        = "system:kube-controller-manager"
	AdminConfSecret  = "admin-kubeconfig"
	CMConfSecret     = "cm-kubeconfig"
	ConfSecretKey    = "kubeconfig"
)

type ConfigGen struct {
	CpName      string
	CpNamespace string
	CpHost      string
	CpPort      int
	Target      ConfigTarget
	caKey       *rsa.PrivateKey
	caTemplate  x509.Certificate
	caPEMCert   []byte
	key         []byte
	cert        []byte
	authInfo    string
	secretName  string
}

func GenerateKubeConfigSecret(ctx context.Context, certs *Certs, conf ConfigGen) (*v1.Secret, error) {
	conf.caKey = certs.caKey
	conf.caTemplate = certs.caTemplate
	conf.caPEMCert = certs.caPEMCert
	if err := conf.generateConfigCerts(); err != nil {
		return nil, err
	}
	kConfig := conf.generateConfig()
	kconf, err := clientcmd.Write(*kConfig)
	if err != nil {
		return nil, err
	}
	return conf.genSecretManifest(ctx, kconf), nil
}

func (c *ConfigGen) genSecretManifest(ctx context.Context, conf []byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.secretName,
			Namespace: c.CpNamespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			ConfSecretKey: conf,
		},
	}
}

func (c *ConfigGen) generateConfigCerts() error {
	var subject pkix.Name
	switch c.Target {
	case Admin:
		subject = pkix.Name{CommonName: AdminCN, Organization: []string{Organization}}
		c.authInfo = GenerateAuthInfoAdminName(c.CpName)
		c.secretName = AdminConfSecret
	case ControllerManager:
		subject = pkix.Name{CommonName: ContrCMCN}
		c.authInfo = ContrCMCN
		c.secretName = CMConfSecret
	default:
		return fmt.Errorf("invalid target: %d", c.Target)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("error generating KubeConfig key pair: %s", err)
	}

	pubKeyHash := sha1.Sum(c.caTemplate.RawSubjectPublicKeyInfo)
	authKeyId := []byte(pubKeyHash[:])
	certTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1658),
		Issuer:                pkix.Name{CommonName: "kubernetes"},
		Subject:               subject,
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		AuthorityKeyId:        authKeyId,
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &c.caTemplate, &key.PublicKey, c.caKey)
	if err != nil {
		return fmt.Errorf("error creating kubeconfig certificate %s", err)
	}
	c.cert = encodeToPEMCertificate(cert)
	c.key = encodeToPEMKey(key)
	return nil
}

func (c *ConfigGen) generateConfig() *clientcmdapi.Config {
	config := clientcmdapi.NewConfig()
	config.Clusters[GenerateClusterName(c.CpName)] = &clientcmdapi.Cluster{
		Server:                   c.generateServerEndpoint(),
		CertificateAuthorityData: c.caPEMCert,
	}
	config.AuthInfos[c.authInfo] = &clientcmdapi.AuthInfo{
		ClientCertificateData: c.cert,
		ClientKeyData:         c.key,
	}
	config.Contexts[GenerateContextName(c.CpName)] = &clientcmdapi.Context{
		Cluster:   GenerateClusterName(c.CpName),
		Namespace: DefaultNamespace,
		AuthInfo:  c.authInfo,
	}
	config.CurrentContext = GenerateContextName(c.CpName)
	return config
}

func (c *ConfigGen) generateServerEndpoint() string {
	return fmt.Sprintf("https://%s", util.GenerateDevLocalDNSName(c.CpName))
}

func GenerateClusterName(cpName string) string {
	return fmt.Sprintf("%s-cluster", cpName)
}

func GenerateAuthInfoAdminName(cpName string) string {
	return fmt.Sprintf("%s-admin", cpName)
}

func GenerateContextName(cpName string) string {
	return cpName
}
