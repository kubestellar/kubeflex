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
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CP struct {
	Ctx        context.Context
	Kubeconfig string
	Name       string
}

func GenerateControlPlane(name, controlPlaneType, backendType, hook string, hookVars []string) *tenancyv1alpha1.ControlPlane {
	cp := &tenancyv1alpha1.ControlPlane{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: tenancyv1alpha1.ControlPlaneSpec{
			Type:    tenancyv1alpha1.ControlPlaneType(controlPlaneType),
			Backend: tenancyv1alpha1.BackendDBType(backendType),
		},
	}
	if hook != "" {
		cp.Spec.PostCreateHook = &hook
		cp.Spec.PostCreateHookVars = convertToMap(hookVars)
	}
	if controlPlaneType == string(tenancyv1alpha1.ControlPlaneTypeExternal) {
		cp.Spec.BootstrapSecretRef = &tenancyv1alpha1.SecretReference{
			Name:         util.GenerateBoostrapSecretName(name),
			Namespace:    util.SystemNamespace,
			InClusterKey: util.KubeconfigSecretKeyInCluster,
		}
	}
	return cp
}

func convertToMap(pairs []string) map[string]string {
	params := make(map[string]string)

	for _, pair := range pairs {
		split := strings.SplitN(pair, "=", 2)
		if len(split) == 2 {
			params[split[0]] = split[1]
		}
	}

	return params
}
