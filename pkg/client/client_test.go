package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"os/user"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	//"github.com/kubestellar/kubeflex/api/v1alpha1"
)

var kubeconfig string

func TestMain(m *testing.M) {
	kubeconfig = os.Getenv("KUBECONFIG")
	user, err := user.Current()
	if err != nil {
		log.Fatalf(err.Error())
	}
	homeDirectory := user.HomeDir
	if kubeconfig == "" {
		kubeconfig = homeDirectory + "/.kube/config"
		if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			// Create a new configuration file with some default contents
			config := clientcmdapi.NewConfig()
			config.Clusters["my-cluster"] = &api.Cluster{
				Server:                   "https://example.com",
				CertificateAuthorityData: []byte(generateTestCA()),
			}
			config.Contexts["my-context"] = &api.Context{
				Cluster:  "my-cluster",
				AuthInfo: "my-user",
			}
			config.AuthInfos["my-user"] = &api.AuthInfo{
				Token: "MY_TOKEN",
			}
			config.CurrentContext = "my-context"
			clientcmd.WriteToFile(*config, kubeconfig)
		}
	}

	code := m.Run()

	os.Exit(code)
}

func TestGetClientSet(t *testing.T) {
	cs := GetClientSet(kubeconfig)
	if cs == nil {
		t.Error("Expected clientset to not be nil")
	}
}

func generateTestCA() string {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1658),
		Subject:               pkix.Name{CommonName: "example.com"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		panic(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	return string(certPEM)
}
