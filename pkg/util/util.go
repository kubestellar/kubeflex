package util

import "fmt"

func GenerateNamespaceFromControlPlaneName(name string) string {
	return fmt.Sprintf("%s-system", name)
}

// GenerateDevLocalDNSName: generates the local dns name for test/dev
// from the controlplane name
func GenerateDevLocalDNSName(name string) string {
	// We use localtest.me for resolving
	return fmt.Sprintf("%s.localtest.me", name)
}
