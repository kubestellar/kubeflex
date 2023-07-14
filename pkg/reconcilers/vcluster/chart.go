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
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	URL         = "https://charts.loft.sh"
	RepoName    = "loft"
	ChartName   = "vcluster"
	ReleaseName = "vcluster"
)

var (
	configs = []string{
		"vcluster.image=rancher/k3s:v1.27.2-k3s1",
	}
)

func (r *VClusterReconciler) ReconcileChart(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	_ = clog.FromContext(ctx)
	localDNSName := util.GenerateDevLocalDNSName(hcp.Name)
	configs = append(configs, fmt.Sprintf("syncer.extraArgs[0]=--tls-san=%s", localDNSName))
	configs = append(configs, fmt.Sprintf("syncer.extraArgs[1]=--out-kube-config-server=https://%s:9443", localDNSName))
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
