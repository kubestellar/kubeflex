/*
Copyright 2024 The KubeStellar Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adopt

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/cmd/kflex/common"
	cont "github.com/kubestellar/kubeflex/cmd/kflex/ctx"
	kfclient "github.com/kubestellar/kubeflex/pkg/client"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CPAdopt struct {
	common.CP
	AdoptedKubeconfig             string
	AdoptedContext                string
	AdoptedURLOverride            string
	AdoptedTokenExpirationSeconds int
	SkipURLOverride               bool
}

// Adopt a control plane from another cluster
func (c *CPAdopt) Adopt(hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.Context(chattyStatus, false, false, false)

	controlPlaneType := tenancyv1alpha1.ControlPlaneTypeExternal
	util.PrintStatus(fmt.Sprintf("Adopting control plane %s of type %s ...", c.Name, controlPlaneType), done, &wg, chattyStatus)

	adoptedKubeconfig := getAdoptedKubeconfig(c)

	clp, err := kfclient.GetClient(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kubeflex client: %v\n", err)
		os.Exit(1)
	}
	cl := *clp

	clientsetp, err := kfclient.GetClientSet(c.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}

	if err := applyAdoptedBootstrapSecret(clientsetp, c.Name, adoptedKubeconfig, c.AdoptedContext, c.AdoptedURLOverride, c.SkipURLOverride); err != nil {
		fmt.Fprintf(os.Stderr, "error creating adopted cluster kubeconfig: %v\n", err)
		os.Exit(1)
	}

	cp := common.GenerateControlPlane(c.Name, string(controlPlaneType), "", hook, hookVars)

	if err := cl.Create(context.TODO(), cp, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ControlPlane object: %v\n", err)
		os.Exit(1)
	}

	done <- true
	wg.Wait()
}

func applyAdoptedBootstrapSecret(clientset *kubernetes.Clientset, cpName, adoptedKubeconfig, contextName, adoptedURLOverride string, skipURLOverride bool) error {
	// Load the kubeconfig from file
	config, err := clientcmd.LoadFromFile(adoptedKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig file %s: %v", adoptedKubeconfig, err)
	}

	// Retrieve the specified context
	context, exists := config.Contexts[contextName]
	if !exists {
		return fmt.Errorf("context %s not found in the kubeconfig", contextName)
	}

	// Retrieve the associated cluster
	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return fmt.Errorf("cluster %s not found for context %s", context.Cluster, contextName)
	}

	// Construct a new kubeConfig object
	kubeConfig := api.NewConfig()

	kubeConfig.Clusters[context.Cluster] = cluster

	if !skipURLOverride {
		// Determine the server endpoint
		endpoint := adoptedURLOverride
		if endpoint == "" {
			endpoint = cluster.Server
			if !isValidServerURL(endpoint) {
				return fmt.Errorf("invalid server endpoint %s. Please provide a valid value with the `url-override` option", endpoint)
			}
		}
		kubeConfig.Clusters[context.Cluster].Server = endpoint
	}

	if authInfo, exists := config.AuthInfos[context.AuthInfo]; exists {
		kubeConfig.AuthInfos[contextName] = authInfo
	} else {
		return fmt.Errorf("authInfo %s not found for context %s", context.AuthInfo, contextName)
	}

	kubeConfig.Contexts[contextName] = &api.Context{
		Cluster:  context.Cluster,
		AuthInfo: contextName,
	}
	kubeConfig.CurrentContext = contextName

	newKubeConfig, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize the new kubeconfig: %v", err)
	}

	createOrUpdateSecret(clientset, cpName, newKubeConfig)

	return nil
}

func createOrUpdateSecret(clientset *kubernetes.Clientset, cpName string, kubeconfig []byte) error {

	// Define the kubeconfig secret
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.GenerateBoostrapSecretName(cpName),
			Namespace: util.SystemNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{util.KubeconfigSecretKeyInCluster: kubeconfig},
	}

	// Try to create the secret
	if _, err := clientset.CoreV1().Secrets(util.SystemNamespace).Create(context.TODO(), kubeConfigSecret, metav1.CreateOptions{}); err != nil {
		// Check if the error is because the secret already exists
		if apierrors.IsAlreadyExists(err) {
			// Retrieve the existing secret
			existingSecret, getErr := clientset.CoreV1().Secrets(util.SystemNamespace).Get(context.TODO(), util.AdminConfSecret, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("failed to fetch existing secret %s in namespace %s: %v", util.AdminConfSecret, util.SystemNamespace, getErr)
			}

			// Update the data of the existing secret
			existingSecret.Data = kubeConfigSecret.Data

			// Update the secret with new data
			if _, updateErr := clientset.CoreV1().Secrets(util.SystemNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{}); updateErr != nil {
				return fmt.Errorf("failed to update existing secret %s in namespace %s: %v", util.AdminConfSecret, util.SystemNamespace, updateErr)
			}
		} else {
			return fmt.Errorf("failed to create secret %s in namespace %s: %v", util.AdminConfSecret, util.SystemNamespace, err)
		}
	}

	return nil
}

// check if the current server URL in the adopted cluster kubeconfig is using
// a local address, which would not work in a container
func isValidServerURL(serverURL string) bool {
	localAddresses := []string{"127.0.0.1", "localhost", "::1"}
	for _, addr := range localAddresses {
		if strings.Contains(serverURL, addr) {
			return false
		}
	}
	return true
}

func getAdoptedKubeconfig(c *CPAdopt) string {
	if c.AdoptedKubeconfig != "" {
		return c.AdoptedKubeconfig
	}
	if c.Kubeconfig != "" {
		return c.Kubeconfig
	}
	return getKubeConfigFromEnv(c.Kubeconfig)
}

func getKubeConfigFromEnv(kubeconfig string) string {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, err := homedir.Dir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
				os.Exit(1)
			}
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
	return kubeconfig
}
