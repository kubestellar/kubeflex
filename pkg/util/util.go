package util

import "fmt"

func GenerateNamespaceFromControlPlaneName(name string) string {
	return fmt.Sprintf("%s-system", name)
}
