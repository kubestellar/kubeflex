package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	// Set up file paths
	certsDir := "./_certs"
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
	kubeconfigPath := filepath.Join(certsDir, "kubeconfig.admin")
	kubeCMconfigPath := filepath.Join(certsDir, "kubeconfig.cm")

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

	// //block, _ := pem.Decode(x509.MarshalPKCS1PrivateKey(caKey))
	// b := x509.MarshalPKCS1PrivateKey(caTemplate.AuthorityKeyId)
	// cert, err := x509.ParseCertificate(b)
	// if err != nil {
	// 	fmt.Printf("Error parsing cert: %v\n", err)
	// 	return
	// }
	pubKeyHash := sha1.Sum(caTemplate.RawSubjectPublicKeyInfo)
	authKeyId := []byte(pubKeyHash[:])
	apiServerTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1658),
		//Subject:      pkix.Name{CommonName: "admin"},
		Subject:               pkix.Name{CommonName: "kube-apiserver"},
		DNSNames:              []string{"kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc.cluster", "kubernetes.default.svc.cluster.local", "localhost"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		AuthorityKeyId:        authKeyId,
		BasicConstraintsValid: true,
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

	// ****************************************************************
	// Generate kubeadmin key and cert
	kubeadminKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating Kubeadmin key pair: %v\n", err)
		return
	}

	kubeAdminTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1658),
		Issuer:                pkix.Name{CommonName: "kubernetes"},
		Subject:               pkix.Name{CommonName: "kubernetes-admin", Organization: []string{"system:masters"}},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		AuthorityKeyId:        authKeyId,
		BasicConstraintsValid: true,
	}
	kubeadminCertBytes, err := x509.CreateCertificate(rand.Reader, &kubeAdminTemplate, &caTemplate, &kubeadminKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating api server CA certificate: %v\n", err)
		return
	}

	// ****************************************************************
	// Generate kubeconfig CM key and cert
	kubeCMKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Error generating KubeCM key pair: %v\n", err)
		return
	}
	kubeCMTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1658),
		Issuer:                pkix.Name{CommonName: "kubernetes"},
		Subject:               pkix.Name{CommonName: "system:kube-controller-manager"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		AuthorityKeyId:        authKeyId,
		BasicConstraintsValid: true,
	}
	kubeCMCertBytes, err := x509.CreateCertificate(rand.Reader, &kubeCMTemplate, &caTemplate, &kubeCMKey.PublicKey, caKey)
	if err != nil {
		fmt.Printf("Error creating CM kubeconfig certificate: %v\n", err)
		return
	}

	// generate admin kubeconfig
	config := clientcmdapi.NewConfig()
	config.Clusters["cp1-cluster"] = &clientcmdapi.Cluster{
		Server:                   "https://localhost:9443",
		CertificateAuthorityData: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes}),
	}
	config.AuthInfos["cp1-admin"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kubeadminCertBytes}),
		ClientKeyData:         pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(kubeadminKey)}),
	}
	config.Contexts["cp1"] = &clientcmdapi.Context{
		Cluster:   "cp1-cluster",
		Namespace: "default",
		AuthInfo:  "cp1-admin",
	}
	config.CurrentContext = "cp1"

	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		panic(err)
	}

	// Write kubeconfig for CM

	cmConfig := clientcmdapi.NewConfig()
	cmConfig.Clusters["kubernetes"] = &clientcmdapi.Cluster{
		Server:                   "https://localhost:9443",
		CertificateAuthorityData: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes}),
	}
	cmConfig.AuthInfos["system:kube-controller-manager"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: kubeCMCertBytes}),
		ClientKeyData:         pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(kubeCMKey)}),
	}
	cmConfig.Contexts["system:kube-controller-manager@kubernetes"] = &clientcmdapi.Context{
		Cluster:   "kubernetes",
		Namespace: "default",
		AuthInfo:  "system:kube-controller-manager",
	}
	cmConfig.CurrentContext = "system:kube-controller-manager@kubernetes"

	if err := clientcmd.WriteToFile(*cmConfig, kubeCMconfigPath); err != nil {
		panic(err)
	}

}
