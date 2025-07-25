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
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"

	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
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

const (
	AdoptedKubeconfigFlag = "adopted-kubeconfig"
	AdoptedContextFlag    = "adopted-context"
	URLOverrideFlag       = "url-override"
	ExpirationSecondsFlag = "expiration-seconds"
)

const defaultExpirationSeconds = 86400 * 365

type CPAdopt struct {
	common.CP
	AdoptedKubeconfig             string
	AdoptedContext                string
	AdoptedURLOverride            string
	AdoptedTokenExpirationSeconds int64
}

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "adopt <name>",
		Short: "Adopt a control plane from an external cluster",
		Long: `Adopt a control plane from an external cluster and switches the Kubeconfig context to
				the current instance`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			flagset := cmd.Flags()
			kubeconfig, _ := flagset.GetString(common.KubeconfigFlag)
			postCreateHook, _ := flagset.GetString(common.PostCreateHookFlag)
			hookVars, _ := flagset.GetStringArray(common.SetFlag)
			chattyStatus, _ := flagset.GetBool(common.ChattyStatusFlag)
			adoptedKubeconfig, _ := flagset.GetString(AdoptedKubeconfigFlag)
			adoptedContext, _ := flagset.GetString(AdoptedContextFlag)
			adoptedURLOverride, _ := flagset.GetString(URLOverrideFlag)
			adoptedTokenExpirationSeconds, _ := flagset.GetInt64(ExpirationSecondsFlag)
			cpAdopt := CPAdopt{
				CP:                            common.NewCP(kubeconfig, common.WithName(args[0])),
				AdoptedKubeconfig:             adoptedKubeconfig,
				AdoptedContext:                adoptedContext,
				AdoptedURLOverride:            adoptedURLOverride,
				AdoptedTokenExpirationSeconds: adoptedTokenExpirationSeconds,
			}
			ExecuteAdopt(cpAdopt, postCreateHook, hookVars, chattyStatus)
		},
	}
	flagset := command.Flags()
	// create passing the control plane type and backend type
	flagset.StringP(common.PostCreateHookFlag, "p", "", "name of post create hook to run")
	flagset.StringArrayP(common.SetFlag, "e", []string{}, "set post create hook variables, in the form name=value ")
	flagset.StringP(AdoptedKubeconfigFlag, "a", "", "path to the kubeconfig file for the adopted cluster. If unspecified, it uses the default Kubeconfig")
	flagset.StringP(AdoptedContextFlag, "c", "", "path to adopted cluster context in adopted kubeconfig")
	flagset.StringP(URLOverrideFlag, "u", "", "URL overrride for adopted cluster. Required when cluster address uses local host address, e.g. `https://127.0.0.1`")
	flagset.Int64P(ExpirationSecondsFlag, "x", defaultExpirationSeconds, "adopted token expiration in seconds. Default is one year.")
	return command
}

// Adopt a control plane from another cluster
func ExecuteAdopt(cpAdopt CPAdopt, hook string, hookVars []string, chattyStatus bool) {
	done := make(chan bool)
	var wg sync.WaitGroup
	cx := cont.CPCtx{}
	cx.ExecuteCtx(chattyStatus, false, false, false)

	controlPlaneType := tenancyv1alpha1.ControlPlaneTypeExternal
	util.PrintStatus(fmt.Sprintf("Adopting control plane %s of type %s ...", cpAdopt.Name, controlPlaneType), done, &wg, chattyStatus)

	cl, err := kfclient.GetClient(cpAdopt.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting kubeflex client: %v\n", err)
		os.Exit(1)
	}

	clientsetp, err := kfclient.GetClientSet(cpAdopt.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting clientset: %v\n", err)
		os.Exit(1)
	}

	if err := applyAdoptedBootstrapSecret(clientsetp, cpAdopt.Name, getBootstrapKubeconfig(cpAdopt), cpAdopt.AdoptedContext, cpAdopt.AdoptedURLOverride); err != nil {
		fmt.Fprintf(os.Stderr, "error creating adopted cluster kubeconfig: %v\n", err)
		os.Exit(1)
	}
	// REFACTOR? is cpAdopt being pointer of (*CPAdopt) make sense?
	controlPlane, err := common.GenerateControlPlane(cpAdopt.Name, string(controlPlaneType), "", hook, hookVars, &cpAdopt.AdoptedTokenExpirationSeconds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating control plane object: %v\n", err)
		os.Exit(1)
	}

	if err := cl.Create(context.TODO(), controlPlane, &client.CreateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ControlPlane object: %v\n", err)
		os.Exit(1)
	}

	done <- true
	wg.Wait()
}

func applyAdoptedBootstrapSecret(clientset *kubernetes.Clientset, cpName, bootstrapKubeconfig, contextName, adoptedURLOverride string) error {
	// Load the kubeconfig from file
	config, err := clientcmd.LoadFromFile(bootstrapKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig file %s: %w", bootstrapKubeconfig, err)
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

	kubeConfig.Clusters[context.Cluster].Server = cluster.Server

	if adoptedURLOverride != "" {
		if err := isValidServerURL(adoptedURLOverride); err != nil {
			return fmt.Errorf("invalid server endpoint %s. Please provide a valid value with the `url-override` option", adoptedURLOverride)
		}
		kubeConfig.Clusters[context.Cluster].Server = adoptedURLOverride
	}

	if authInfo, exists := config.AuthInfos[context.AuthInfo]; exists {
		kubeConfig.AuthInfos[contextName] = authInfo
	} else {
		return fmt.Errorf("authInfo %s not found for context %s", context.AuthInfo, contextName)
	}

	kubeConfig.Contexts[contextName] = &api.Context{
		Cluster:    context.Cluster,
		AuthInfo:   contextName,
		Extensions: context.Extensions,
		Namespace:  context.Namespace,
	}
	kubeConfig.CurrentContext = contextName

	newKubeConfig, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize the new kubeconfig: %w", err)
	}

	return createOrUpdateSecret(clientset, cpName, newKubeConfig)
}

func createOrUpdateSecret(clientset *kubernetes.Clientset, cpName string, kubeconfig []byte) error {

	bootstrapSecretName := util.GenerateBootstrapSecretName(cpName)

	// Define the kubeconfig secret
	kubeConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstrapSecretName,
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
			existingSecret, getErr := clientset.CoreV1().Secrets(util.SystemNamespace).Get(context.TODO(), bootstrapSecretName, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("failed to fetch existing secret %s in namespace %s: %w", util.AdminConfSecret, bootstrapSecretName, getErr)
			}

			// Update the data of the existing secret
			existingSecret.Data = kubeConfigSecret.Data

			// Update the secret with new data
			if _, updateErr := clientset.CoreV1().Secrets(util.SystemNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{}); updateErr != nil {
				return fmt.Errorf("failed to update existing secret %s in namespace %s: %w", bootstrapSecretName, util.SystemNamespace, updateErr)
			}
		} else {
			return fmt.Errorf("failed to create secret %s in namespace %s: %w", bootstrapSecretName, util.SystemNamespace, err)
		}
	}

	return nil
}

// check if the current server URL in the adopted cluster kubeconfig is a valid URL
// and it not using a local address, which would not work in a container
func isValidServerURL(serverURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Ensure the URL scheme is either http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("URL must start with http:// or https://")
	}

	// Ensure the host is non-empty
	if parsedURL.Host == "" {
		return errors.New("URL must have a host part")
	}

	// Reject URLs with user information (i.e., username or password)
	if parsedURL.User != nil {
		return errors.New("URL must not contain user info")
	}

	// Reject URLs containing query parameters
	if parsedURL.RawQuery != "" {
		return errors.New("URL must not contain query parameters")
	}

	// Reject URLs containing fragments
	if parsedURL.Fragment != "" {
		return errors.New("URL must not contain fragments")
	}

	localAddresses := []string{"127.0.0.1", "localhost", "::1"}
	for _, addr := range localAddresses {
		if parsedURL.Host == addr {
			return fmt.Errorf("URL must not use addresses in %v", localAddresses)
		}
	}
	return nil
}

func getBootstrapKubeconfig(cpAdopt CPAdopt) string {
	if cpAdopt.AdoptedKubeconfig != "" {
		return cpAdopt.AdoptedKubeconfig
	}
	if cpAdopt.Kubeconfig != "" {
		return cpAdopt.Kubeconfig
	}
	return getKubeConfigFromEnv()
}

func getKubeConfigFromEnv() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	return kubeconfig
}
