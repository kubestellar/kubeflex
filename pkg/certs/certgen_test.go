package certs

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	extraDNSNames := []string{"example.com"}

	// Test that New returns a non-nil Certs struct
	certs, err := New(ctx, extraDNSNames)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if certs == nil {
		t.Fatal("New returned nil certs")
	}

	// Test that all certs were generated successfully
	if certs.caPEMCert == nil || certs.caKey == nil {
		t.Error("caCert or caKey was not generated")
	}
	if certs.apiServerPEMCert == nil || certs.apiServerPEMKey == nil {
		t.Error("apiServerCert or apiServerKey was not generated")
	}
	if certs.kubeletPEMCert == nil || certs.kubeletPEMKey == nil {
		t.Error("kubeletCert or kubeletKey was not generated")
	}
	if certs.frontProxyPEMCert == nil || certs.frontProxyPEMKey == nil {
		t.Error("frontProxyCert or frontProxyKey was not generated")
	}
	if certs.saPEMPubKey == nil || certs.saPEMKey == nil {
		t.Error("saCert or saKey was not generated")
	}
}

func TestCerts_generateCA(t *testing.T) {
	c := Certs{}
	ctx := context.Background()
	err := c.generateCA(ctx)
	if err != nil {
		t.Errorf("Error returned from generateCA function: %v", err)
	}
	if c.caKey == nil || c.caTemplate.SerialNumber == nil || c.caPEMKey == nil || c.caPEMCert == nil {
		t.Error("generateCA did not properly generate all necessary fields")
	}
}

func TestCerts_generateAPIServerKeyAndCert(t *testing.T) {
	c := Certs{}
	ctx := context.Background()
	err := c.generateCA(ctx)
	if err != nil {
		t.Fatalf("Error generating CA in order to create API server key and cert: %v", err)
	}
	extraDNSNames := []string{"example.com"}
	err = c.generateAPIServerKeyAndCert(ctx, extraDNSNames)
	if err != nil {
		t.Errorf("Error returned from generateAPIServerKeyAndCert function: %v", err)
	}
	if c.apiServerPEMCert == nil || c.apiServerPEMKey == nil {
		t.Error("generateAPIServerKeyAndCert did not properly generate PEM certificates and keys")
	}
}

func TestCerts_generateKubeletKeyAndCert(t *testing.T) {
	c := Certs{}
	ctx := context.Background()
	err := c.generateCA(ctx)
	if err != nil {
		t.Fatalf("Error generating CA in order to create kubelet key and cert: %v", err)
	}
	err = c.generateKubeletKeyAndCert(ctx)
	if err != nil {
		t.Errorf("Error returned from generateKubeletKeyAndCert function: %v", err)
	}
	if c.kubeletPEMCert == nil || c.kubeletPEMKey == nil {
		t.Error("generateKubeletKeyAndCert did not properly generate PEM certificates and keys")
	}
}

func TestCerts_generateFrontProxyKeyAndCert(t *testing.T) {
	c := Certs{}
	ctx := context.Background()
	err := c.generateCA(ctx)
	if err != nil {
		t.Fatalf("Error generating CA in order to create front proxy key and cert: %v", err)
	}
	err = c.generateFrontProxyKeyAndCert(ctx)
	if err != nil {
		t.Errorf("Error returned from generateFrontProxyKeyAndCert function: %v", err)
	}
	if c.frontProxyPEMCert == nil || c.frontProxyPEMKey == nil {
		t.Error("generateFrontProxyKeyAndCert did not properly generate PEM certificates and keys")
	}
}

func TestCerts_generateSAKey(t *testing.T) {
	c := Certs{}
	ctx := context.Background()
	err := c.generateSAKey(ctx)
	if err != nil {
		t.Errorf("Error returned from generateSAKey function: %v", err)
	}
	if c.saPEMKey == nil || c.saPEMPubKey == nil {
		t.Error("generateSAKey did not properly generate PEM keys")
	}
}
