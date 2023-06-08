package client

import (
	"log"
	"os"
	"os/user"
	"testing"

	"mcc.ibm.org/kubeflex/api/v1alpha1"
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
		os.Setenv("KUBECONFIG", homeDirectory+"/.kube/config")
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

func TestGetClient(t *testing.T) {
	c := GetClient(kubeconfig)
	if c == nil {
		t.Error("Expected client to not be nil")
	}

	// Make sure the custom type has been added to the scheme
	x := *c
	scheme := x.Scheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Errorf("Failed to add custom type to scheme: %v", err)
	}
}
