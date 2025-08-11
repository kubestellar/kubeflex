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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
	"errors"
)

const (
	// FieldManager is the field manager name used when applying Kubernetes resources
	// via server-side apply. This identifies kubeflex as the manager of specific fields.
	FieldManager = "kubeflex"
)

// ErrPostCreateHookNotFound is returned when a referenced post-create hook
// cannot be found in the cluster. This typically indicates a configuration error
// where a ControlPlane references a non-existent PostCreateHook resource.
var ErrPostCreateHookNotFound = errors.New("post create hook not found")

// Vars represents the system variables that are automatically injected into
// post-create hook templates. These variables provide context about the current
// control plane and hook being processed.
type Vars struct {
	// Namespace is the Kubernetes namespace where the control plane resources are created
	Namespace string
	// ControlPlaneName is the name of the ControlPlane resource being processed
	ControlPlaneName string
	// HookName is the name of the PostCreateHook being executed
	HookName string
}

// ReconcileUpdatePostCreateHook is the main orchestrator that processes all post-create hooks
// for a given ControlPlane resource. It implements conditional completion logic based on the
// WaitForPostCreateHooks flag, which determines whether the controller should wait for
// resources to be ready before marking hooks as complete.
//
// The function processes hooks in the following order:
// 1. Legacy hook from Spec.PostCreateHook (for backward compatibility)
// 2. New hooks from Spec.PostCreateHooks array (in declared order)
//
// Hook processing includes:
// - Variable resolution with proper precedence (defaults -> global -> user -> system)
// - Resource application via server-side apply
// - Optional readiness checking based on WaitForPostCreateHooks setting
// - Status updates to track completion state
//
// Returns an error if any critical failures occur during processing.
func (r *BaseReconciler) ReconcileUpdatePostCreateHook(ctx context.Context, hcp *v1alpha1.ControlPlane) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Collect all hooks to process (legacy + new) while preserving order
	hooks := make([]v1alpha1.PostCreateHookUse, 0)
	seen := make(map[string]bool)

	// Add legacy hook first if specified (backward compatibility)
	// This ensures existing configurations continue to work
	if hcp.Spec.PostCreateHook != nil && *hcp.Spec.PostCreateHook != "" {
		hookName := *hcp.Spec.PostCreateHook
		hooks = append(hooks, v1alpha1.PostCreateHookUse{
			HookName: hcp.Spec.PostCreateHook,
			Vars:     hcp.Spec.PostCreateHookVars,
		})
		seen[hookName] = true
	}

	// Add new hooks in declared order, skipping duplicates
	// This prevents the same hook from being applied multiple times
	for _, hook := range hcp.Spec.PostCreateHooks {
		if hook.HookName != nil && *hook.HookName != "" {
			hookName := *hook.HookName
			if !seen[hookName] {
				hooks = append(hooks, hook)
				seen[hookName] = true
			} else {
				logger.Info("Skipping duplicate hook", "hook", hookName)
			}
		}
	}

	var errs []error
	allHooksApplied := true
	allResourcesReady := true

	// Process each hook
	for _, hook := range hooks {
		hookName := *hook.HookName

		// Check if hook is already applied
		// This allows for idempotent reconciliation - hooks won't be reapplied unnecessarily
		_, ok := hcp.Status.PostCreateHooks[hookName]
		if ok {
			// Hook is applied, check if resources are ready only if WaitForPostCreateHooks is enabled
			if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
				logger.Info("Checking post-create hook resources", "hook", hookName)
				if err := r.checkPostCreateHookResources(ctx, hcp, hookName, hook.Vars); err != nil {
					logger.Info("Hook resources not ready", "hook", hookName, "error", err)
					allResourcesReady = false
				}
			}
			continue
		}

		logger.Info("Processing post-create hook", "hook", hookName)

		// Get hook definition from cluster
		pch := &v1alpha1.PostCreateHook{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, pch); err != nil {
			return fmt.Errorf("%w: %v", ErrPostCreateHookNotFound, err)
		}

		// Build variables with precedence: defaults -> global -> user vars -> system
		// This precedence ensures system variables cannot be overridden while allowing
		// flexibility for user customization
		vars := make(map[string]interface{})

		// 1. Default vars from hook spec (lowest priority)
		for _, dv := range pch.Spec.DefaultVars {
			vars[dv.Name] = dv.Value
		}

		// 2. Global variables from control plane
		for key, val := range hcp.Spec.GlobalVars {
			vars[key] = val
		}

		// 3. User-provided vars from hook use
		for key, val := range hook.Vars {
			vars[key] = val
		}

		// 4. System variables (highest priority - cannot be overridden)
		vars["Namespace"] = namespace
		vars["ControlPlaneName"] = hcp.Name
		vars["HookName"] = hookName

		// Apply hook templates to the cluster
		appliedResources, err := r.applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, pch, vars, hcp)
		if err != nil {
			if util.IsTransientError(err) {
				// Transient errors (network issues, temporary API unavailability) should be retried
				errs = append(errs, fmt.Errorf("transient error applying hook %s: %w", hookName, err))
				allHooksApplied = false
			} else {
				// Permanent errors (invalid templates, missing CRDs) should be logged and tracked
				logger.Error(err, "Permanent error applying post-create hook", "hook", hookName)
				errs = append(errs, fmt.Errorf("permanent error applying hook %s: %w", hookName, err))
				allHooksApplied = false
			}
			continue
		}

		// Check if newly applied resources are ready (if enabled)
		// This is only done for newly applied resources to avoid redundant checks
		if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
			ready, err := r.checkAppliedResourcesReady(ctx, appliedResources, namespace)
			if err != nil {
				if util.IsTransientError(err) {
					errs = append(errs, fmt.Errorf("transient error checking readiness for hook %s: %w", hookName, err))
				} else {
					logger.Error(err, "Error checking resource readiness for hook", "hook", hookName)
					errs = append(errs, fmt.Errorf("error checking readiness for hook %s: %w", hookName, err))
				}
				allHooksApplied = false
				allResourcesReady = false
				continue
			}
			if !ready {
				logger.Info("Resources not ready yet for hook", "hook", hookName)
				allResourcesReady = false
				// Don't mark hook as applied yet - will retry on next reconciliation
				continue
			}
		}

		// Update status - mark hook as applied
		// This tracking allows for idempotent reconciliation and progress visibility
		if hcp.Status.PostCreateHooks == nil {
			hcp.Status.PostCreateHooks = make(map[string]bool)
		}
		hcp.Status.PostCreateHooks[hookName] = true
		if err := r.Client.Status().Update(ctx, hcp); err != nil {
			errs = append(errs, fmt.Errorf("failed to update status for hook %s: %w", hookName, err))
			allHooksApplied = false
			// Break on status update error to prevent inconsistent state
			break
		}

		// Propagate labels from hook to control plane
		// This allows hooks to influence the control plane's metadata for organizational purposes
		if err := r.propagateLabels(pch, hcp, r.Client); err != nil {
			logger.Error(err, "Failed to propagate labels from hook", "hook", hookName)
		}
	}

	// Update PostCreateHookCompleted status based on configuration
	if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
		// NEW BEHAVIOR: Complete only when all hooks applied AND all resources ready
		// This provides stronger guarantees about system readiness
		hcp.Status.PostCreateHookCompleted = allHooksApplied && allResourcesReady && len(errs) == 0
	} else {
		// OLD BEHAVIOR: Complete when all hooks applied (don't wait for resources)
		// This maintains backward compatibility for faster deployments
		hcp.Status.PostCreateHookCompleted = allHooksApplied && len(errs) == 0
	}

	// Update final status
	if err := r.Client.Status().Update(ctx, hcp); err != nil {
		errs = append(errs, fmt.Errorf("failed to update final status: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors processing hooks: %v", errs)
	}
	return nil
}

// checkPostCreateHookResources checks if all resources created by a specific already-applied hook are ready.
// This function is used during reconciliation to verify that previously applied hooks maintain their
// ready state, which is important when WaitForPostCreateHooks is enabled.
//
// The function reconstructs the same variable context that was used during initial application
// to ensure consistency in template rendering and resource identification.
//
// Returns an error if any resource is not ready or if there are issues accessing the resources.
func (r *BaseReconciler) checkPostCreateHookResources(ctx context.Context, hcp *v1alpha1.ControlPlane, hookName string, hookVars map[string]string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Get the post create hook definition
	hook := &v1alpha1.PostCreateHook{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, hook); err != nil {
		return fmt.Errorf("error retrieving post create hook %s: %w", hookName, err)
	}

	// Build variables with proper precedence (same as main function)
	// This ensures we check the same resources that were actually created
	vars := make(map[string]interface{})

	// Add default variables from hook spec
	for _, dv := range hook.Spec.DefaultVars {
		vars[dv.Name] = dv.Value
	}

	// Add global variables from control plane
	for key, val := range hcp.Spec.GlobalVars {
		vars[key] = val
	}

	// Override with user-provided variables from hook use
	for key, val := range hookVars {
		vars[key] = val
	}

	// Add system variables (highest priority)
	vars["Namespace"] = namespace
	vars["ControlPlaneName"] = hcp.Name
	vars["HookName"] = hookName

	// Check each template's resource status
	for _, template := range hook.Spec.Templates {
		// Render the template with variables
		rendered, err := util.RenderYAML(template.Raw, vars)
		if err != nil {
			return fmt.Errorf("error rendering template: %w", err)
		}

		obj, err := util.ToUnstructured(rendered)
		if err != nil {
			return fmt.Errorf("error converting rendered template to unstructured: %w", err)
		}

		gvk := util.GetGroupVersionKindFromObject(obj)
		gvr, err := util.GVKToGVR(r.ClientSet, gvk)
		if err != nil {
			return fmt.Errorf("error getting GVR: %w", err)
		}

		// Check resource readiness
		ready, err := r.checkResourceStatus(ctx, gvr, obj.GetName(), namespace, gvk.Kind)
		if err != nil {
			return fmt.Errorf("error checking resource status: %w", err)
		}

		if !ready {
			return fmt.Errorf("resource %s/%s not ready", gvk.Kind, obj.GetName())
		}
	}

	return nil
}

// ResourceInfo holds information about an applied resource for readiness checking.
// This structure captures the minimal information needed to later verify if a
// resource is ready, without requiring full resource objects to be stored.
type ResourceInfo struct {
	// Name is the Kubernetes resource name
	Name string
	// Namespace is the Kubernetes namespace (empty for cluster-scoped resources)
	Namespace string
	// GVR is the GroupVersionResource used to access the resource via dynamic client
	GVR schema.GroupVersionResource
	// Kind is the Kubernetes resource kind (e.g., "Deployment", "Service")
	Kind string
	// IsClusterScoped indicates whether the resource is cluster-scoped or namespaced
	IsClusterScoped bool
}

// applyPostCreateHook applies all templates in a hook and returns info about applied resources.
// This function is responsible for the actual resource creation/update in the cluster using
// server-side apply for idempotent operations.
//
// The function processes each template by:
// 1. Rendering the template with the provided variables
// 2. Converting to unstructured objects for dynamic application
// 3. Determining the appropriate scope (cluster vs namespace)
// 4. Applying via server-side apply with field management
// 5. Tracking applied resources for potential readiness checking
//
// Returns a slice of ResourceInfo for later readiness verification and any errors encountered.
func (r *BaseReconciler) applyPostCreateHook(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, hook *v1alpha1.PostCreateHook, vars map[string]interface{}, hcp *v1alpha1.ControlPlane) ([]ResourceInfo, error) {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Get API resource information for scope determination
	apiResourceLists, err := clientSet.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	var appliedResources []ResourceInfo

	// Process each template in the hook
	for _, template := range hook.Spec.Templates {
		// Render template with variable substitution
		rendered, err := util.RenderYAML(template.Raw, vars)
		if err != nil {
			return nil, err
		}

		// Convert rendered YAML to unstructured object for dynamic processing
		obj, err := util.ToUnstructured(rendered)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			return nil, fmt.Errorf("null object in template")
		}

		// Determine resource type information
		gvk := util.GetGroupVersionKindFromObject(obj)
		gvr, err := util.GVKToGVR(clientSet, gvk)
		if err != nil {
			return nil, err
		}

		// Determine if resource is cluster-scoped or namespaced
		clusterScoped, err := util.IsClusterScoped(gvk, apiResourceLists)
		if err != nil {
			return nil, err
		}

		logger.Info("Applying", "object", util.GenerateObjectInfoString(*obj), "cpNamespace", namespace)

		// Apply the resource using server-side apply for idempotent operations
		if clusterScoped {
			// Add tracking labels for cluster-scoped resources
			r.setTrackingLabelsAndAnnotations(obj, hcp.Name)
			_, err = dynamicClient.Resource(gvr).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
			// Apply to specific namespace for namespaced resources
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		}
		if err != nil {
			return nil, err
		}

		// Track applied resource for readiness checking
		resourceInfo := ResourceInfo{
			Name:            obj.GetName(),
			Namespace:       namespace,
			GVR:             gvr,
			Kind:            gvk.Kind,
			IsClusterScoped: clusterScoped,
		}
		appliedResources = append(appliedResources, resourceInfo)
	}

	return appliedResources, nil
}

// checkAppliedResourcesReady checks if all newly applied resources are ready.
// This function iterates through a list of ResourceInfo objects and verifies
// that each resource has reached a ready state according to its specific readiness criteria.
//
// Returns true only if ALL resources are ready, false if any resource is not ready,
// and an error if there are issues checking resource status.
func (r *BaseReconciler) checkAppliedResourcesReady(ctx context.Context, resources []ResourceInfo, namespace string) (bool, error) {
	for _, resource := range resources {
		ready, err := r.checkResourceStatus(ctx, resource.GVR, resource.Name, namespace, resource.Kind)
		if err != nil {
			return false, err
		}
		if !ready {
			return false, nil
		}
	}
	return true, nil
}

// checkResourceStatus checks if a specific resource is ready based on its type.
// This function implements type-specific readiness logic for various Kubernetes resources.
// It uses a proven approach with minimal essential additions to handle the most common
// resource types that require actual readiness verification.
//
// Readiness criteria by resource type:
// - Deployment/StatefulSet: All replicas are ready
// - Job: At least one pod has succeeded
// - DaemonSet: All desired pods are ready
// - CustomResourceDefinition: CRD is established and available
// - Pod: Pod is in Running or Succeeded phase
// - Others: Resource exists (immediate readiness)
//
// Returns true if the resource is ready, false if not ready, and an error for access issues.
func (r *BaseReconciler) checkResourceStatus(ctx context.Context, gvr schema.GroupVersionResource, name, namespace, kind string) (bool, error) {
	switch kind {
	// Core workload resources that need actual readiness checking
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment); err != nil {
			return false, err
		}
		// Handle nil replicas (Kubernetes defaults to 1)
		expectedReplicas := int32(1)
		if deployment.Spec.Replicas != nil {
			expectedReplicas = *deployment.Spec.Replicas
		}
		return deployment.Status.ReadyReplicas == expectedReplicas, nil

	case "StatefulSet":
		statefulset := &appsv1.StatefulSet{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, statefulset); err != nil {
			return false, err
		}
		// Handle nil replicas (Kubernetes defaults to 1)
		expectedReplicas := int32(1)
		if statefulset.Spec.Replicas != nil {
			expectedReplicas = *statefulset.Spec.Replicas
		}
		return statefulset.Status.ReadyReplicas == expectedReplicas, nil

	case "Job":
		job := &batchv1.Job{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, job); err != nil {
			return false, err
		}
		// Job is ready when at least one pod has completed successfully
		return job.Status.Succeeded > 0, nil

	case "DaemonSet":
		daemonset := &appsv1.DaemonSet{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, daemonset); err != nil {
			return false, err
		}
		// DaemonSet is ready when all desired pods are ready
		return daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled, nil

	// Only add CRD support if hooks actually create them
	case "CustomResourceDefinition":
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
			return false, err
		}
		// Check if CRD is established and available for use
		for _, condition := range crd.Status.Conditions {
			if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil

	// Add Pod support for standalone pods (less common but possible)
	case "Pod":
		pod := &corev1.Pod{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, pod); err != nil {
			return false, err
		}
		// Pod is ready when running or has succeeded (for job-like pods)
		return pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded, nil

	default:
		// For everything else (Services, ConfigMaps, Secrets, etc.)
		// Just check if they exist - they're ready immediately when created
		// These resources don't have complex startup sequences
		if namespace != "" {
			_, err := r.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				// Try cluster-scoped if namespaced fails (fallback for misclassified resources)
				_, err = r.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
			}
			return err == nil, nil
		} else {
			// Cluster-scoped resource
			_, err := r.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
			return err == nil, nil
		}
	}
}

// setTrackingLabelsAndAnnotations sets labels and annotations used by Helm for garbage collection
// and resource tracking. This ensures that resources created by post-create hooks can be properly
// managed and cleaned up when the control plane is removed.
//
// The function adds:
// - ManagedBy label set to "Helm" for integration with Helm's lifecycle management
// - Release namespace annotation for proper scoping and cleanup
//
// This is only applied to cluster-scoped resources since namespaced resources are automatically
// cleaned up when their namespace is deleted.
func (r *BaseReconciler) setTrackingLabelsAndAnnotations(obj *unstructured.Unstructured, cpName string) {
	namespace := util.GenerateNamespaceFromControlPlaneName(cpName)

	// Add or update labels for Helm management
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[util.ManagedByKey] = "Helm"
	obj.SetLabels(labels)

	// Add or update annotations for proper namespace tracking
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[util.HelmReleaseNamespaceAnnotationKey] = namespace
	obj.SetAnnotations(annotations)
}

// propagateLabels copies labels from PostCreateHook to ControlPlane for consistency.
// This allows hooks to influence the control plane's metadata, which can be useful for
// organizational purposes, policy enforcement, or integration with other systems.
//
// The function only updates the ControlPlane if there are actual changes to avoid
// unnecessary API calls and potential conflicts.
//
// Returns an error if the update operation fails.
func (r *BaseReconciler) propagateLabels(hook *v1alpha1.PostCreateHook, hcp *v1alpha1.ControlPlane, c client.Client) error {
	hookLabels := hook.GetLabels()
	if len(hookLabels) == 0 {
		return nil
	}

	hcpLabels := hcp.GetLabels()
	if hcpLabels == nil {
		hcpLabels = map[string]string{}
	}

	// Check if any labels need to be added or updated
	updateRequired := false
	for key, value := range hookLabels {
		if existingValue, exists := hcpLabels[key]; !exists || existingValue != value {
			updateRequired = true
			hcpLabels[key] = value
		}
	}

	// Only perform update if changes are needed
	if updateRequired {
		hcp.SetLabels(hcpLabels)
		if err := c.Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}