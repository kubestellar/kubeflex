package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Set up file paths
	certsDir := "./certs"
	caCrtPath := filepath.Join(certsDir, "ca.crt")
	caKeyPath := filepath.Join(certsDir, "ca.key")
	apiServerKubeletClientCrtPath := filepath.Join(certsDir, "apiserver-kubelet-client.crt")
	apiServerKubeletClientKeyPath := filepath.Join(certsDir, "apiserver-kubelet-client.key")
	frontProxyClientCrtPath := filepath.Join(certsDir, "front-proxy-client.crt")
	frontProxyClientKeyPath := filepath.Join(certsDir, "front-proxy-client.key")
	frontProxyCaCrtPath := filepath.Join(certsDir, "front-proxy-ca.crt")
	saPubPath := filepath.Join(certsDir, "sa.pub")
	saKeyPath := filepath.Join(certsDir, "sa.key")
	apiServerCrtPath := filepath.Join(certsDir, "apiserver.crt")
	apiServerKeyPath := filepath.Join(certsDir, "apiserver.key")

	// Create certs directory if it doesn't exist
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		err = os.MkdirAll(certsDir, 0700)
		if err != nil {
			fmt.Printf("Error creating certs directory: %v\n", err)
			return
		}
	}

	// Generate CA certificate
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating CA key: %v\n", err)
		return
	}
	caTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1654),
		Subject:               pkix.Name{CommonName: "My CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating CA certificate: %v\n", err)
		return
	}
	err = os.WriteFile(caCrtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing CA certificate file: %v\n", err)
		return
	}
	err = os.WriteFile(caKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)}), 0600)
	if err != nil {
		fmt.Printf("Error writing CA key file: %v\n", err)
		return
	}

	// Generate Kubelet client key and certificate
	kubeletClientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating Kubelet client key: %v\n", err)
		return
	}
	kubeletClientTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1655),
		Subject:               pkix.Name{CommonName: "apiserver-kubelet-client"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	kubeletClientBytes, err := x509.CreateCertificate(rand.Reader, &kubeletClientTemplate, &caTemplate, &kubeletClientKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating Kubelet client certificate: %v\n", err)
		return
	}
	err = os.WriteFile(apiServerKubeletClientCrtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kubeletClientBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing Kubelet client certificate file: %v\n", err)
		return
	}
	err = os.WriteFile(apiServerKubeletClientKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(kubeletClientKey)}), 0600)
	if err != nil {
		fmt.Printf("Error writing Kubelet client key file: %v\n", err)
		return
	}

	// Generate front proxy client key and certificate
	frontProxyClientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating front proxy client key: %v\n", err)
		return
	}
	frontProxyClientTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1656),
		Subject:               pkix.Name{CommonName: "front-proxy-client"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	frontProxyClientBytes, err := x509.CreateCertificate(rand.Reader, &frontProxyClientTemplate, &caTemplate, &frontProxyClientKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating front proxy client certificate: %v\n", err)
		return
	}
	err = os.WriteFile(frontProxyClientCrtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: frontProxyClientBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing front proxy client certificate file: %v\n", err)
		return
	}
	err = os.WriteFile(frontProxyClientKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(frontProxyClientKey)}), 0600)
	if err != nil {
		fmt.Printf("Error writing front proxy client key file: %v\n", err)
		return
	}

	// Generate front proxy CA certificate
	frontProxyCaTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1657),
		Subject:               pkix.Name{CommonName: "My Front Proxy CA"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	frontProxyCaBytes, err := x509.CreateCertificate(rand.Reader, &frontProxyCaTemplate, &caTemplate, &frontProxyClientKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating front proxy CA certificate: %v\n", err)
		return
	}
	err = os.WriteFile(frontProxyCaCrtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: frontProxyCaBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing front proxy CA certificate file: %v\n", err)
		return
	}

	// Generate service account key pair
	serviceAccountKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating service account key pair: %v\n", err)
		return
	}
	serviceAccountPubBytes, err := x509.MarshalPKIXPublicKey(&serviceAccountKey.PublicKey)
	if err != nil {
		fmt.Printf("Error marshalling service account public key: %v\n", err)
		return
	}
	err = os.WriteFile(saPubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: serviceAccountPubBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing service account public key file: %v\n", err)
		return
	}
	err = os.WriteFile(saKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serviceAccountKey)}), 0600)
	if err != nil {
		fmt.Printf("Error writing service account private key file: %v\n", err)
		return
	}

	// Generate API server TLS key pair
	apiServerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating API server TLS key pair: %v\n", err)
		return
	}
	apiServerTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1658),
		Subject:               pkix.Name{CommonName: "kubernetes"},
		DNSNames:              []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster", "kubernetes.default.svc.cluster.local"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	apiServerCaBytes, err := x509.CreateCertificate(rand.Reader, &apiServerTemplate, &caTemplate, &apiServerKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating api server CA certificate: %v\n", err)
		return
	}
	err = os.WriteFile(apiServerCrtPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: apiServerCaBytes}), 0644)
	if err != nil {
		fmt.Printf("Error writing api server certificate file: %v\n", err)
		return
	}
	err = os.WriteFile(apiServerKeyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(apiServerKey)}), 0600)
	if err != nil {
		fmt.Printf("Error writing api server key file: %v\n", err)
		return
	}
}
