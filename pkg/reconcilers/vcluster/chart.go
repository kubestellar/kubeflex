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

package vcluster

import (
	"context"
	"fmt"
	"strings"

	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/helm"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	URL         = "https://charts.loft.sh"
	Version     = "0.16.4"
	RepoName    = "loft"
	ChartName   = "vcluster"
	ReleaseName = "vcluster"
)

var (
	configsBase = []string{
		"vcluster.image=rancher/k3s:v1.27.2-k3s1",
	}
)

func (r *VClusterReconciler) ReconcileChart(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, cfg *shared.SharedConfig) error {
	_ = clog.FromContext(ctx)
	dnsName := util.GenerateDevLocalDNSName(hcp.Name, cfg.Domain)
	port := cfg.ExternalPort
	configs := append([]string{}, configsBase...)
	if cfg.ExternalURL != "" {
		dnsName = cfg.ExternalURL
		port = 443
		// TODO - this is specific to OpenShift, not to having an external URL - ok for now, but to improve later
		//configs = append(configs, "openshift.enable=true")
		ocpConfigs := []string{
			"openshift.enable=true",
			"securityContext.allowPrivilegeEscalation=false",
			"securityContext.capabilities.drop={ALL}",
			"securityContext.runAsUser=null",
			"securityContext.runAsGroup=null",
			"securityContext.seccompProfile.type=RuntimeDefault",
		}
		configs = append(configs, ocpConfigs...)
	}
	configs = append(configs, fmt.Sprintf("syncer.extraArgs[0]=--tls-san=%s", dnsName))
	configs = append(configs, fmt.Sprintf("syncer.extraArgs[1]=--out-kube-config-server=https://%s:%d", dnsName, port))
	configs = append(configs, fmt.Sprintf("syncer.extraArgs[2]=--tls-san=%s", cfg.HostContainer))
	h := &helm.HelmHandler{
		URL:         URL,
		RepoName:    RepoName,
		ChartName:   ChartName,
		Version:     Version,
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
