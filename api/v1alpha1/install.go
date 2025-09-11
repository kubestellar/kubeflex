package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	AddToScheme(scheme)
}
