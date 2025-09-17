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

package shared

import (
	"context"
	"fmt"

	"github.com/kubestellar/kubeflex/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

func (r *BaseReconciler) ReconcileNamespace(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	log := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)
	// create namespace object
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	log.Info("ReconcileNamespace to create", "ns", ns)
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(ns), ns, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := controllerutil.SetControllerReference(hcp, ns, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			if err = r.Client.Create(ctx, ns, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return err
				}
				return fmt.Errorf("failed to create namespace: %w", err)
			}
		} else if util.IsTransientError(err) {
			return err
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}
