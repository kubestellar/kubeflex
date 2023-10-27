package util

import (
	"context"

	"github.com/kubestellar/kubeflex/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func AddSCCtoUserPolicy(kubeconfig string) error {
	securityClient, err := client.GetOpendShiftSecClient("")
	if err != nil {
		return err
	}

	anyuidSCC, err := securityClient.SecurityV1().SecurityContextConstraints().Get(context.Background(), "anyuid", metav1.GetOptions{})
	if err != nil {
		return err
	}

	anyuidSCC.Users = append(anyuidSCC.Users, kubeflexServiceAccount)

	_, err = securityClient.SecurityV1().SecurityContextConstraints().Update(context.Background(), anyuidSCC, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
