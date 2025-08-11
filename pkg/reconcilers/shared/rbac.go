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
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

const (
	// roleName is the standard name used for the cluster info updater role and role binding.
	// This role provides the necessary RBAC permissions for jobs and services that need to
	// update cluster information, including managing deployments, services, and secrets
	// within the control plane namespace.
	roleName = "cluster-info-updater"
)

// ReconcileUpdateClusterInfoJobRole ensures that the required RBAC Role exists for cluster info jobs.
// This function manages the lifecycle of a Kubernetes Role that grants permissions necessary
// for cluster information update operations within the control plane namespace.
//
// The role provides permissions to:
// - Manage deployments and statefulsets (apps API group)
// - Manage services (core API group)  
// - Manage secrets (core API group)
//
// The function follows the standard controller reconciliation pattern:
// 1. Check if the role already exists
// 2. If not found, create it with proper ownership references
// 3. Handle transient errors with retry logic
// 4. Return permanent errors without retry
//
// The role is automatically garbage collected when the parent ControlPlane is deleted
// due to the controller reference that is established.
//
// Returns an error if the role cannot be created or if there are persistent API issues.
func (r *BaseReconciler) ReconcileUpdateClusterInfoJobRole(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Create role object template for existence check
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
	}

	// Check if role already exists
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(role), role, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Role doesn't exist, create it
			role := generateClusterInfoJobRole(roleName, namespace)
			
			// Set controller reference for automatic garbage collection
			// This ensures the role is deleted when the ControlPlane is removed
			if err := controllerutil.SetControllerReference(hcp, role, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			
			// Create the role in the cluster
			if err = r.Client.Create(ctx, role, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					// Network issues, temporary API unavailability, etc. should be retried
					return err // Retry transient errors
				}
				// Permanent errors like validation failures, insufficient permissions
				return fmt.Errorf("failed to create role: %w", err)
			}
		} else if util.IsTransientError(err) {
			// Transient error during Get operation (network, API server issues)
			return err // Retry transient errors
		} else {
			// Permanent error during Get operation (permissions, invalid request)
			return fmt.Errorf("failed to get role: %w", err)
		}
	}
	// Role exists and is accessible - no action needed
	return nil
}

// generateClusterInfoJobRole creates a new Role object with the necessary permissions
// for cluster information update operations. This function defines the specific RBAC
// rules that allow services to manage cluster-related resources.
//
// The generated role includes three main permission sets:
//
// 1. Apps API Group (deployments, statefulsets):
//    - get, watch, list: Read access for monitoring and discovery
//    - create, update: Write access for managing workload resources
//
// 2. Core API Group - Services:
//    - get, list: Read access to discover existing services
//    - create, update: Write access to manage service endpoints and configuration
//
// 3. Core API Group - Secrets:
//    - get, list: Read access to retrieve configuration and credentials
//    - create, update: Write access to store updated cluster information securely
//
// The permissions are scoped to the namespace level, providing security isolation
// between different control plane instances.
//
// Parameters:
//   - name: The name to assign to the Role resource
//   - namespace: The Kubernetes namespace where the Role should be created
//
// Returns a configured Role object ready for creation in the cluster.
func generateClusterInfoJobRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				// Permissions for managing application workloads
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets"},
				Verbs:     []string{"get", "watch", "list", "create", "update"},
			},
			{
				// Permissions for managing network services
				APIGroups: []string{""}, // Core API group (empty string)
				Resources: []string{"services"},
				Verbs:     []string{"get", "list", "create", "update"},
			},
			{
				// Permissions for managing sensitive configuration data
				APIGroups: []string{""}, // Core API group (empty string)
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "update"},
			},
		},
	}
}

// ReconcileUpdateClusterInfoJobRoleBinding ensures that the required RBAC RoleBinding exists
// to associate the cluster-info-updater Role with the appropriate ServiceAccount.
// This function completes the RBAC setup by binding the permissions defined in the Role
// to a specific identity (ServiceAccount) that can exercise those permissions.
//
// The RoleBinding connects:
// - Role: cluster-info-updater (with the permissions for managing cluster resources)
// - Subject: default ServiceAccount in the control plane namespace
//
// This setup allows jobs, pods, or other workloads running under the default ServiceAccount
// in the control plane namespace to perform cluster information update operations.
//
// The function follows the same reconciliation pattern as the Role creation:
// 1. Check if the RoleBinding already exists
// 2. If not found, create it with proper ownership references
// 3. Handle transient vs permanent errors appropriately
//
// The RoleBinding is automatically cleaned up when the parent ControlPlane is deleted
// due to the established controller reference.
//
// Returns an error if the RoleBinding cannot be created or if there are API access issues.
func (r *BaseReconciler) ReconcileUpdateClusterInfoJobRoleBinding(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Create role binding object template for existence check
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
	}

	// Check if role binding already exists
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(binding), binding, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// RoleBinding doesn't exist, create it
			binding := generateClusterInfoJobRoleBinding(roleName, namespace)
			
			// Set controller reference for automatic garbage collection
			// This ensures the RoleBinding is deleted when the ControlPlane is removed
			if err := controllerutil.SetControllerReference(hcp, binding, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			
			// Create the role binding in the cluster
			if err = r.Client.Create(ctx, binding, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					// Network issues, temporary API unavailability should be retried
					return err // Retry transient errors
				}
				// Permanent errors like validation failures, role doesn't exist, insufficient permissions
				return fmt.Errorf("failed to create role binding: %w", err)
			}
		} else if util.IsTransientError(err) {
			// Transient error during Get operation
			return err // Retry transient errors
		} else {
			// Permanent error during Get operation
			return fmt.Errorf("failed to get role binding: %w", err)
		}
	}
	// RoleBinding exists and is accessible - no action needed
	return nil
}

// generateClusterInfoJobRoleBinding creates a new RoleBinding object that associates
// the cluster-info-updater Role with the default ServiceAccount in the specified namespace.
// This binding is the final piece that enables workloads to exercise the permissions
// defined in the corresponding Role.
//
// The RoleBinding structure:
//
// RoleRef: References the Role that defines the permissions
// - APIGroup: rbac.authorization.k8s.io (standard RBAC API group)
// - Kind: Role (indicates this is a namespace-scoped role, not ClusterRole)
// - Name: The name of the Role to bind to (same as the RoleBinding name)
//
// Subjects: Specifies who gets the permissions (the identity being granted access)
// - Kind: ServiceAccount (Kubernetes service identity for pods/jobs)
// - Name: "default" (the default ServiceAccount that exists in every namespace)
// - Namespace: The control plane namespace (ensures proper scoping)
//
// This approach uses the default ServiceAccount for simplicity, but in production
// environments, you might want to create dedicated ServiceAccounts for better
// security isolation and auditability.
//
// Parameters:
//   - name: The name to assign to the RoleBinding resource (matches the Role name)
//   - namespace: The Kubernetes namespace where the RoleBinding should be created
//
// Returns a configured RoleBinding object ready for creation in the cluster.
func generateClusterInfoJobRoleBinding(name, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			// Reference to the Role that defines the permissions
			APIGroup: rbacv1.GroupName, // rbac.authorization.k8s.io
			Kind:     "Role",           // Namespace-scoped role
			Name:     name,             // Must match the Role name exactly
		},
		Subjects: []rbacv1.Subject{
			{
				// Grant permissions to the default ServiceAccount
				Kind:      "ServiceAccount",
				Name:      "default", // Default ServiceAccount in the namespace
				Namespace: namespace, // Must be in the same namespace as the Role
			},
		},
	}
}