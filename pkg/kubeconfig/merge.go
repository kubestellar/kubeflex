package kubeconfig

import (
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func merge(existing, new *clientcmdapi.Config) error {
	for k, v := range new.Clusters {
		existing.Clusters[k] = v
	}

	for k, v := range new.AuthInfos {
		existing.AuthInfos[k] = v
	}

	for k, v := range new.Contexts {
		existing.Contexts[k] = v
	}

	// set the current context
	existing.CurrentContext = new.CurrentContext
	return nil
}
