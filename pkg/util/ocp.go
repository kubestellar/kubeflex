package util

import (
	"k8s.io/client-go/kubernetes"
)

func IsOpenShift(clientset kubernetes.Clientset) bool {
	return CheckResourceExists(clientset, "project.openshift.io", "v1", "projects")
}
