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

	"errors"
	"github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	FieldManager = "kubeflex"
)

var ErrPostCreateHookNotFound = errors.New("post create hook not found")

type Vars struct {
	Namespace        string
	ControlPlaneName string
	HookName         string
}

// ReconcileUpdatePostCreateHook is the main orchestrator that processes all post-create hooks
// and implements conditional completion logic based on WaitForPostCreateHooks flag
func (r *BaseReconciler) ReconcileUpdatePostCreateHook(ctx context.Context, hcp *v1alpha1.ControlPlane) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Collect all hooks to process (legacy + new) while preserving order
	hooks := make([]v1alpha1.PostCreateHookUse, 0)
	seen := make(map[string]bool)

	// Add legacy hook first if specified (backward compatibility)
	if hcp.Spec.PostCreateHook != nil && *hcp.Spec.PostCreateHook != "" {
		hookName := *hcp.Spec.PostCreateHook
		hooks = append(hooks, v1alpha1.PostCreateHookUse{
			HookName: hcp.Spec.PostCreateHook,
			Vars:     hcp.Spec.PostCreateHookVars,
		})
		seen[hookName] = true
	}

	// Add new hooks in declared order, skipping duplicates
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

		// Get hook definition
		pch := &v1alpha1.PostCreateHook{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, pch); err != nil {
			return fmt.Errorf("%w: %v", ErrPostCreateHookNotFound, err)
		}

		// Build variables with precedence: defaults -> global -> user vars -> system
		vars := make(map[string]interface{})

		// 1. Default vars from hook spec
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

		// 4. System variables (highest priority)
		vars["Namespace"] = namespace
		vars["ControlPlaneName"] = hcp.Name
		vars["HookName"] = hookName

		// Apply hook templates
		appliedResources, err := r.applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, pch, vars, hcp)
		if err != nil {
			if util.IsTransientError(err) {
				errs = append(errs, fmt.Errorf("transient error applying hook %s: %w", hookName, err))
				allHooksApplied = false
			} else {
				logger.Error(err, "Permanent error applying post-create hook", "hook", hookName)
				errs = append(errs, fmt.Errorf("permanent error applying hook %s: %w", hookName, err))
				allHooksApplied = false
			}
			continue
		}

		// Check if newly applied resources are ready (if enabled)
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
				// Don't mark hook as applied yet - will retry
				continue
			}
		}

		// Update status - mark hook as applied
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
		if err := r.propagateLabels(pch, hcp, r.Client); err != nil {
			logger.Error(err, "Failed to propagate labels from hook", "hook", hookName)
		}
	}

	// Update PostCreateHookCompleted status
	if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
		// NEW BEHAVIOR: Complete only when all hooks applied AND all resources ready
		hcp.Status.PostCreateHookCompleted = allHooksApplied && allResourcesReady && len(errs) == 0
	} else {
		// OLD BEHAVIOR: Complete when all hooks applied (don't wait for resources)
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

// checkPostCreateHookResources checks if all resources created by a specific already-applied hook are ready
func (r *BaseReconciler) checkPostCreateHookResources(ctx context.Context, hcp *v1alpha1.ControlPlane, hookName string, hookVars map[string]string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Get the post create hook definition
	hook := &v1alpha1.PostCreateHook{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, hook); err != nil {
		return fmt.Errorf("error retrieving post create hook %s: %w", hookName, err)
	}

	// Build variables with proper precedence (same as main function)
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

// ResourceInfo holds information about an applied resource for readiness checking
type ResourceInfo struct {
	Name            string
	Namespace       string
	GVR             schema.GroupVersionResource
	Kind            string
	IsClusterScoped bool
}

// applyPostCreateHook applies all templates in a hook and returns info about applied resources
func (r *BaseReconciler) applyPostCreateHook(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, hook *v1alpha1.PostCreateHook, vars map[string]interface{}, hcp *v1alpha1.ControlPlane) ([]ResourceInfo, error) {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	apiResourceLists, err := clientSet.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	var appliedResources []ResourceInfo

	for _, template := range hook.Spec.Templates {
		// Render template
		rendered, err := util.RenderYAML(template.Raw, vars)
		if err != nil {
			return nil, err
		}

		obj, err := util.ToUnstructured(rendered)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			return nil, fmt.Errorf("null object in template")
		}

		gvk := util.GetGroupVersionKindFromObject(obj)
		gvr, err := util.GVKToGVR(clientSet, gvk)
		if err != nil {
			return nil, err
		}

		clusterScoped, err := util.IsClusterScoped(gvk, apiResourceLists)
		if err != nil {
			return nil, err
		}

		logger.Info("Applying", "object", util.GenerateObjectInfoString(*obj), "cpNamespace", namespace)

		// Apply the resource
		if clusterScoped {
			r.setTrackingLabelsAndAnnotations(obj, hcp.Name)
			_, err = dynamicClient.Resource(gvr).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
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

// checkAppliedResourcesReady checks if all newly applied resources are ready
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

// checkResourceStatus checks if a specific resource is ready based on its type
// This uses the proven approach from the old code with minimal essential additions
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
		return job.Status.Succeeded > 0, nil

	case "DaemonSet":
		daemonset := &appsv1.DaemonSet{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, daemonset); err != nil {
			return false, err
		}
		return daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled, nil

	// Only add CRD support if hooks actually create them
	case "CustomResourceDefinition":
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
			return false, err
		}
		// Check if CRD is established
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
		return pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded, nil

	default:
		// For everything else (Services, ConfigMaps, Secrets, etc.)
		// Just check if they exist - they're ready immediately when created
		if namespace != "" {
			_, err := r.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				// Try cluster-scoped if namespaced fails
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

// setTrackingLabelsAndAnnotations sets labels used by helm for garbage collection
func (r *BaseReconciler) setTrackingLabelsAndAnnotations(obj *unstructured.Unstructured, cpName string) {
	namespace := util.GenerateNamespaceFromControlPlaneName(cpName)

	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[util.ManagedByKey] = "Helm"
	obj.SetLabels(labels)

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[util.HelmReleaseNamespaceAnnotationKey] = namespace
	obj.SetAnnotations(annotations)
}

// propagateLabels copies labels from PostCreateHook to ControlPlane for consistency
func (r *BaseReconciler) propagateLabels(hook *v1alpha1.PostCreateHook, hcp *v1alpha1.ControlPlane, c client.Client) error {
	hookLabels := hook.GetLabels()
	if len(hookLabels) == 0 {
		return nil
	}

	hcpLabels := hcp.GetLabels()
	if hcpLabels == nil {
		hcpLabels = map[string]string{}
	}

	updateRequired := false
	for key, value := range hookLabels {
		if existingValue, exists := hcpLabels[key]; !exists || existingValue != value {
			updateRequired = true
			hcpLabels[key] = value
		}
	}

	if updateRequired {
		hcp.SetLabels(hcpLabels)
		if err := c.Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
