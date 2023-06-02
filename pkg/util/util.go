package util

import "fmt"

const (
	DBReleaseName = "postgres"
	DBNamespace   = "kubeflex-system"
)

func GenerateNamespaceFromControlPlaneName(name string) string {
	return fmt.Sprintf("%s-system", name)
}

// GenerateDevLocalDNSName: generates the local dns name for test/dev
// from the controlplane name
func GenerateDevLocalDNSName(name string) string {
	// We use localtest.me for resolving
	return fmt.Sprintf("%s.localtest.me", name)
}

func GeneratePSecretName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}

func GeneratePSReplicaSetName(releaseName string) string {
	return fmt.Sprintf("%s-postgresql", releaseName)
}
