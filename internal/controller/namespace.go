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

package controller

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"mcc.ibm.org/kubeflex/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ControlPlaneReconciler) ReconcileNamespace(ctx context.Context, name string, owner *metav1.OwnerReference) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(name)

	// create namespace object
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(ns), ns, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			util.EnsureOwnerRef(ns, owner)
			err = r.Client.Create(context.TODO(), ns, &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}
