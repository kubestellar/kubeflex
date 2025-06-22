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

package ocm

import (
	"context"
	"fmt"
	"strings"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/helm"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	URL         = "oci://quay.io/kubestellar/multicluster-controlplane-chart:v0.2.0-kf-alpha.1"
	RepoName    = "multicluster-controlplane"
	ChartName   = "multicluster-controlplane-chart"
	ReleaseName = "multicluster-controlplane"
)

var (
	configs = []string{
		"image=quay.io/kubestellar/multicluster-controlplane:v0.2.0-kf-alpha.1",
		"route.enabled=false",
		"apiserver.internalHostname=kubeflex-control-plane",
		"nodeport.enabled=false",
	}
)

func (r *OCMReconciler) ReconcileChart(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, cfg *shared.SharedConfig) error {
	dnsName := util.GenerateDevLocalDNSName(hcp.Name, cfg.Domain)
	port := cfg.ExternalPort
	if cfg.ExternalURL != "" {
		dnsName = cfg.ExternalURL
		port = shared.DefaultPort
	}
	configs = append(configs, fmt.Sprintf("apiserver.externalHostname=%s", dnsName))
	configs = append(configs, fmt.Sprintf("apiserver.port=%d", port))
	h := &helm.HelmHandler{
		URL:         URL,
		RepoName:    RepoName,
		ChartName:   ChartName,
		Namespace:   util.GenerateNamespaceFromControlPlaneName(hcp.Name),
		ReleaseName: ReleaseName,
		Args:        map[string]string{"set": strings.Join(configs, ",")},
	}
	err := helm.Init(ctx, h)
	if err != nil {
		return err
	}

	if !h.IsDeployed() {
		err := h.Install()
		if err != nil {
			return err
		}
	}
	return nil
}
