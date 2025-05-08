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

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

type CP struct {
	Ctx        context.Context
	Kubeconfig string
	Name       string
	AliasName  string
}

type CPOption func(*CP)

// New control plane with option design pattern
func NewCP(kubeconfigPath string, options ...CPOption) CP {
	cp := CP{Ctx: createContext(), Kubeconfig: kubeconfigPath}
	for _, option := range options {
		option(&cp)
	}
	return cp
}

// Make Name optional using option design pattern
func WithName(name string) CPOption {
	return func(cp *CP) {
		cp.Name = name
	}
}

// Make AliasName optional using option design pattern
func WithAliasName(aliasName string) CPOption {
	return func(cp *CP) {
		cp.AliasName = aliasName
	}
}

// Create a new system context with logger
func createContext() context.Context {
	zapLogger, _ := zap.NewDevelopment(zap.AddCaller())
	logger := zapr.NewLoggerWithOptions(zapLogger)
	return logr.NewContext(context.Background(), logger)
}

// REFACTOR: the usage of the variable `cp` as *common.CP and then overshadowed by `cp` as tenancyv1alpha1.ControlPlane was tricky. Took a while to understand. Hence, `cp` from tenancyv1alpha1 becomes `controlPlane` and common.CP is referred by `cp`
func GenerateControlPlane(name, controlPlaneType, backendType, hook string, hookVars []string, tokenExpirationSeconds *int64) (*tenancyv1alpha1.ControlPlane, error) {
	controlPlane := &tenancyv1alpha1.ControlPlane{
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
		controlPlane.Spec.PostCreateHook = &hook
		var err error
		controlPlane.Spec.PostCreateHookVars, err = convertToMap(hookVars)
		if err != nil {
			return nil, err
		}
	}
	if controlPlaneType == string(tenancyv1alpha1.ControlPlaneTypeExternal) {
		controlPlane.Spec.BootstrapSecretRef = &tenancyv1alpha1.BootstrapSecretReference{
			Name:         util.GenerateBootstrapSecretName(name),
			Namespace:    util.SystemNamespace,
			InClusterKey: util.KubeconfigSecretKeyInCluster,
		}
	}
	return controlPlane, nil
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
