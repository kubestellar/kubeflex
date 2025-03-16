/*
Copyright 2023 The KubeStellar Authors.

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

package common

import (
	"context"
	"fmt"
	"strings"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
	"github.com/kubestellar/kubeflex/pkg/client"
)

type CP struct {
	Ctx        context.Context
	Kubeconfig string
	Name       string
}

func GenerateControlPlane(name, controlPlaneType, backendType, hook string, hookVars []string, tokenExpirationSeconds *int64) (*tenancyv1alpha1.ControlPlane, error) {
	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: tenancyv1alpha1.ControlPlaneSpec{
			Type:                   tenancyv1alpha1.ControlPlaneType(controlPlaneType),
			Backend:                tenancyv1alpha1.BackendDBType(backendType),
			TokenExpirationSeconds: tokenExpirationSeconds,
		},
	}
	if hook != "" {
		cp.Spec.PostCreateHook = &hook
		var err error
		cp.Spec.PostCreateHookVars, err = convertToMap(hookVars)
		if err != nil {
			return nil, err
		}
	}
	if controlPlaneType == string(tenancyv1alpha1.ControlPlaneTypeExternal) {
		cp.Spec.BootstrapSecretRef = &tenancyv1alpha1.BootstrapSecretReference{
			Name:         util.GenerateBootstrapSecretName(name),
			Namespace:    util.SystemNamespace,
			InClusterKey: util.KubeconfigSecretKeyInCluster,
		}
	}
	return cp, nil
}

func convertToMap(pairs []string) (map[string]string, error) {
	params := make(map[string]string)

	for _, pair := range pairs {
		// Ensure the pair is not empty
		if pair == "" {
			continue
		}

		// Split the pair into key and value using "=" as the delimiter
		split := strings.SplitN(pair, "=", 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("unexpected expression %q. Must be in the form 'key=value'", pair)
		}

		key := strings.TrimSpace(split[0])
		value := strings.TrimSpace(split[1])

		if key == "" {
			return nil, fmt.Errorf("invalid key in expression %q", pair)
		}

		params[key] = value
	}

	return params, nil
}

func (cp *CP) List(chattyStatus bool) {
	clientset, err := client.GetClientSet(cp.Kubeconfig)
	if err != nil {
		fmt.Printf("Error getting clientset: %s\n", err)
		os.Exit(1)
	}

	cps, err := clientset.CoreV1().CustomResourceDefinitions().List(cp.Ctx, metav1.ListOptions{
		LabelSelector: "tenancy.kflex.kubestellar.io/controlplane",
	})
	if err != nil {
		fmt.Printf("Error listing control planes: %s\n", err)
		os.Exit(1)
	}

	if len(cps.Items) == 0 {
		fmt.Println("No control planes found.")
		return
	}

	fmt.Println("Control Planes:")
	for _, cp := range cps.Items {
		fmt.Printf("- %s\n", cp.Name)
	}
}