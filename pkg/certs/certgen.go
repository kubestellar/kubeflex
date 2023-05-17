package certs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

type Certs struct {
	caKey             *rsa.PrivateKey
	caTemplate        x509.Certificate
	caPEMKey          []byte
	caPEMCert         []byte
	apiServerPEMKey   []byte
	apiServerPEMCert  []byte
	kubeletPEMKey     []byte
	kubeletPEMCert    []byte
	frontProxyPEMKey  []byte
	frontProxyPEMCert []byte
	saPEMKey          []byte
	saPEMPubKey       []byte
}

func New(ctx context.Context) (*Certs, error) {
	c := &Certs{}
	if err := c.generateAllCerts(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Certs) generateAllCerts(ctx context.Context) error {
	if err := c.generateCA(ctx); err != nil {
		return err
	}
	if err := c.generateAPIServerKeyAndCert(ctx); err != nil {
		return err
	}
	if err := c.generateKubeletKeyAndCert(ctx); err != nil {
		return err
	}
	if err := c.generateFrontProxyKeyAndCert(ctx); err != nil {
		return err
	}
	if err := c.generateSAKey(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Certs) generateCA(ctx context.Context) (err error) {
	log := clog.FromContext(ctx)
	c.caKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error generating CA key")
		return err
	}

	c.caTemplate = x509.Certificate{
		SerialNumber: big.NewInt(1654),
		Subject: pkix.Name{
			Organization:       []string{"Kubernetes"},
			OrganizationalUnit: []string{"API Server"},
			CommonName:         "kubernetes",
		},
		Issuer:                pkix.Name{CommonName: "kubernetes"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, &c.caTemplate, &c.caTemplate, &c.caKey.PublicKey, c.caKey)
	if err != nil {
		log.Error(err, "Error creating CA certificate: %v\n")
		return err
	}
	c.caPEMKey = encodeToPEMKey(c.caKey)
	c.caPEMCert = encodeToPEMCertificate(caBytes)
	return nil
}

func (c *Certs) generateAPIServerKeyAndCert(ctx context.Context) (err error) {
	log := clog.FromContext(ctx)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error generating API server TLS key pair")
		return err
	}

	pubKeyHash := sha1.Sum(c.caTemplate.RawSubjectPublicKeyInfo)
	authKeyId := []byte(pubKeyHash[:])
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject:      pkix.Name{CommonName: "kube-apiserver"},
		DNSNames: []string{"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster",
			"kubernetes.default.svc.cluster.local",
			"localhost"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		AuthorityKeyId:        authKeyId,
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &c.caTemplate, &key.PublicKey, c.caKey)
	if err != nil {
		log.Error(err, "Error creating api server certificate")
		return err
	}
	c.apiServerPEMCert = encodeToPEMCertificate(cert)
	c.apiServerPEMKey = encodeToPEMKey(key)
	return nil
}

func (c *Certs) generateKubeletKeyAndCert(ctx context.Context) (err error) {
	log := clog.FromContext(ctx)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error generating kubelet key")
		return err
	}

	certTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1655),
		Subject:               pkix.Name{CommonName: "apiserver-kubelet-client"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &c.caTemplate, &key.PublicKey, c.caKey)
	if err != nil {
		log.Error(err, "Error creating kubelet certificate")
		return err
	}
	c.kubeletPEMCert = encodeToPEMCertificate(cert)
	c.kubeletPEMKey = encodeToPEMKey(key)
	return nil
}

func (c *Certs) generateFrontProxyKeyAndCert(ctx context.Context) (err error) {
	log := clog.FromContext(ctx)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error generating front proxy key")
		return err
	}

	certTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1656),
		Subject:               pkix.Name{CommonName: "front-proxy-client"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &c.caTemplate, &key.PublicKey, c.caKey)
	if err != nil {
		log.Error(err, "Error creating front proxy certificate")
		return err
	}
	c.frontProxyPEMCert = encodeToPEMCertificate(cert)
	c.frontProxyPEMKey = encodeToPEMKey(key)
	return nil
}

func (c *Certs) generateSAKey(ctx context.Context) (err error) {
	log := clog.FromContext(ctx)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Error(err, "Error generating service account key pair")
		return err
	}
	pubKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		log.Error(err, "Error marshalling service account public key")
		return err
	}
	c.saPEMKey = encodeToPEMKey(key)
	c.saPEMPubKey = encodeToPEMPublicKey(pubKey)
	return nil
}

func encodeToPEMCertificate(cert []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
}

func encodeToPEMKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}

func encodeToPEMPublicKey(key []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: key})
}
