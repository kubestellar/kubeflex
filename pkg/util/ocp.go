package util

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

const (
	kubeflexServiceAccount = "system:serviceaccount:kubeflex-system:kubeflex-controller-manager"
)

func IsOpenShift(clientset kubernetes.Clientset) bool {
	return CheckResourceExists(clientset, "project.openshift.io", "v1", "Project")
}

func CheckResourceExists(clientset kubernetes.Clientset, group, version, kind string) bool {
	discoveryClient := clientset.Discovery()

	exists, _ := checkResourceExists(discoveryClient, group, version, kind)

	return exists
}

func checkResourceExists(discoveryClient discovery.DiscoveryInterface, group, version, kind string) (bool, error) {
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(group + "/" + version)
	if err != nil {
		return false, err
	}

	for _, resource := range resourceList.APIResources {
		if resource.Kind == kind {
			return true, nil
		}
	}

	return false, nil
}
