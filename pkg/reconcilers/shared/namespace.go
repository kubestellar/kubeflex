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

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

// ReconcileNamespace ensures that the Kubernetes Namespace associated with the
// provided ControlPlane exists. If the Namespace does not exist, it is created
// and set as a child of the ControlPlane resource.
//
// Parameters:
//   - ctx: Context for request-scoped deadlines and cancellation.
//   - hcp: The ControlPlane resource whose namespace should be reconciled.
//
// Behaviour:
//   - Generates a namespace name based on the ControlPlane name.
//   - Checks if the namespace exists.
//   - Creates the namespace if it does not exist.
//   - Retries on transient errors.
func (r *BaseReconciler) ReconcileNamespace(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	// Generate the namespace name from the ControlPlane name.
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Define the Namespace resource object.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	// Attempt to fetch the existing Namespace.
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(ns), ns, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Set the ControlPlane as the namespace's owner for garbage collection.
			if err := controllerutil.SetControllerReference(hcp, ns, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			// Create the Namespace resource.
			if err = r.Client.Create(ctx, ns, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return err // Retry transient errors.
				}
				return fmt.Errorf("failed to create namespace: %w", err)
			}
		} else if util.IsTransientError(err) {
			return err // Retry transient errors.
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}
