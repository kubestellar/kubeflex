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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/util"
)

const (
	FieldManager = "kubeflex"
)

type Vars struct {
	Namespace        string
	ControlPlaneName string
	HookName         string
}

func (r *BaseReconciler) ReconcileUpdatePostCreateHook(ctx context.Context, hcp *v1alpha1.ControlPlane) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Collect all hooks to process (legacy + new) while preserving order
	hooks := make([]v1alpha1.PostCreateHookUse, 0)
	seen := make(map[string]bool)

	// Add legacy hook first if specified
	if hcp.Spec.PostCreateHook != nil {
		hookName := *hcp.Spec.PostCreateHook
		hooks = append(hooks, v1alpha1.PostCreateHookUse{
			HookName: hcp.Spec.PostCreateHook,
			Vars:     hcp.Spec.PostCreateHookVars,
		})
		seen[hookName] = true
	}

	// Add new hooks in declared order, skipping duplicates
	for _, hook := range hcp.Spec.PostCreateHooks {
		if hook.HookName != nil {
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

	for _, hook := range hooks {
		hookName := *hook.HookName

		// Skip already applied hooks
		if hcp.Status.PostCreateHooks != nil && hcp.Status.PostCreateHooks[hookName] {
			continue
		}

		logger.Info("Processing post-create hook", "hook", hookName)

		// Get hook definition
		pch := &v1alpha1.PostCreateHook{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, pch); err != nil {
			errs = append(errs, fmt.Errorf("failed to get PostCreateHook %s: %w", hookName, err))
			allHooksApplied = false
			continue
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

		// 4. System variables
		vars["Namespace"] = namespace
		vars["ControlPlaneName"] = hcp.Name
		vars["HookName"] = hookName

		// Apply hook templates
		appliedResources, err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, pch, vars, hcp)
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

		// Check if resources are ready (if enabled)
		if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
			ready, err := checkResourcesReady(ctx, r.ClientSet, r.DynamicClient, appliedResources, namespace)
			if err != nil {
				if util.IsTransientError(err) {
					errs = append(errs, fmt.Errorf("transient error checking readiness for hook %s: %w", hookName, err))
				} else {
					logger.Error(err, "Error checking resource readiness for hook", "hook", hookName)
					errs = append(errs, fmt.Errorf("error checking readiness for hook %s: %w", hookName, err))
				}
				allHooksApplied = false
				continue
			}
			if !ready {
				logger.Info("Resources not ready yet for hook", "hook", hookName)
				allHooksApplied = false
				continue
			}
		}

		// Update status
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

		// Propagate labels
		if err := propagateLabels(pch, hcp, r.Client); err != nil {
			logger.Error(err, "Failed to propagate labels from hook", "hook", hookName)
		}
	}

	// Update final "PostCreateHookCompleted" flag if all hooks are processed successfully
	if allHooksApplied && len(errs) == 0 {
		if !hcp.Status.PostCreateHookCompleted {
			hcp.Status.PostCreateHookCompleted = true
			if err := r.Client.Status().Update(ctx, hcp); err != nil {
				errs = append(errs, fmt.Errorf("failed to update PostCreateHookCompleted status: %w", err))
			} else {
				logger.Info("All post-create hooks completed successfully")
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors processing hooks: %v", errs)
	}
	return nil
}

// ResourceInfo holds information about an applied resource for readiness checking
type ResourceInfo struct {
	Name            string
	Namespace       string
	GVR             schema.GroupVersionResource
	IsClusterScoped bool
}

func applyPostCreateHook(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, hook *v1alpha1.PostCreateHook, vars map[string]interface{}, hcp *v1alpha1.ControlPlane) ([]ResourceInfo, error) {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)
	apiResourceLists, err := clientSet.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	var appliedResources []ResourceInfo

	for _, template := range hook.Spec.Templates {
		raw := template.Raw
		rendered, err := util.RenderYAML(raw, vars)
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

		if clusterScoped {
			setTrackingLabelsAndAnnotations(obj, hcp.Name)
			_, err = dynamicClient.Resource(gvr).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		}
		if err != nil {
			return nil, err
		}

		// Add to applied resources for readiness checking
		resourceInfo := ResourceInfo{
			Name:            obj.GetName(),
			Namespace:       namespace,
			GVR:             gvr,
			IsClusterScoped: clusterScoped,
		}
		if !clusterScoped {
			resourceInfo.Namespace = namespace
		}
		appliedResources = append(appliedResources, resourceInfo)
	}
	return appliedResources, nil
}

// checkResourcesReady checks if all applied resources are ready
func checkResourcesReady(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, resources []ResourceInfo, namespace string) (bool, error) {
	for _, resource := range resources {
		ready, err := isResourceReady(ctx, clientSet, dynamicClient, resource)
		if err != nil {
			return false, err
		}
		if !ready {
			return false, nil
		}
	}
	return true, nil
}

// isResourceReady checks if a specific resource is ready based on its type
func isResourceReady(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, resource ResourceInfo) (bool, error) {
	var obj *unstructured.Unstructured
	var err error

	if resource.IsClusterScoped {
		obj, err = dynamicClient.Resource(resource.GVR).Get(ctx, resource.Name, metav1.GetOptions{})
	} else {
		obj, err = dynamicClient.Resource(resource.GVR).Namespace(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
	}

	if err != nil {
		return false, err
	}

	// Check readiness based on resource type
	switch resource.GVR.Resource {
	case "deployments":
		return isDeploymentReady(obj)
	case "statefulsets":
		return isStatefulSetReady(obj)
	case "daemonsets":
		return isDaemonSetReady(obj)
	case "jobs":
		return isJobReady(obj)
	case "pods":
		return isPodReady(obj)
	case "services":
		// Services are typically ready immediately
		return true, nil
	case "configmaps", "secrets":
		// ConfigMaps and Secrets are ready when they exist
		return true, nil
	case "customresourcedefinitions":
		return isCRDReady(obj)
	default:
		// For unknown resource types, check if they have a ready condition
		return hasReadyCondition(obj), nil
	}
}

func isDeploymentReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return false, err
	}

	replicas, found, err := unstructured.NestedInt64(spec, "replicas")
	if err != nil || !found {
		replicas = 1 // Default replicas
	}

	readyReplicas, _, _ := unstructured.NestedInt64(status, "readyReplicas")
	availableReplicas, _, _ := unstructured.NestedInt64(status, "replicas")

	return readyReplicas == replicas && availableReplicas == replicas && replicas > 0, nil
}

func isStatefulSetReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return false, err
	}

	replicas, found, err := unstructured.NestedInt64(spec, "replicas")
	if err != nil || !found {
		replicas = 1 // Default replicas
	}

	readyReplicas, _, _ := unstructured.NestedInt64(status, "readyReplicas")
	availableReplicas, _, _ := unstructured.NestedInt64(status, "replicas")

	return readyReplicas == replicas && availableReplicas == replicas && replicas > 0, nil
}

func isDaemonSetReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	desiredNumberScheduled, _, _ := unstructured.NestedInt64(status, "desiredNumberScheduled")
	numberReady, _, _ := unstructured.NestedInt64(status, "numberReady")

	return desiredNumberScheduled > 0 && numberReady == desiredNumberScheduled, nil
}

func isJobReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	// Check if job has completed successfully
	conditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		return false, err
	}

	for _, condition := range conditions {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		condStatus, _, _ := unstructured.NestedString(condMap, "status")

		if condType == "Complete" && condStatus == "True" {
			return true, nil
		}
		if condType == "Failed" && condStatus == "True" {
			return false, fmt.Errorf("job failed")
		}
	}

	return false, nil
}

func isPodReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	phase, _, _ := unstructured.NestedString(status, "phase")
	if phase == "Running" || phase == "Succeeded" {
		// Also check ready condition
		conditions, found, err := unstructured.NestedSlice(status, "conditions")
		if err != nil || !found {
			return phase == "Succeeded", nil // If no conditions, consider Succeeded phase as ready
		}

		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(condMap, "type")
			condStatus, _, _ := unstructured.NestedString(condMap, "status")

			if condType == "Ready" && condStatus == "True" {
				return true, nil
			}
		}
	}

	return false, nil
}

func isCRDReady(obj *unstructured.Unstructured) (bool, error) {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return false, err
	}

	conditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		return false, err
	}

	for _, condition := range conditions {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		condStatus, _, _ := unstructured.NestedString(condMap, "status")

		if condType == "Established" && condStatus == "True" {
			return true, nil
		}
	}

	return false, nil
}

func hasReadyCondition(obj *unstructured.Unstructured) bool {
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if err != nil || !found {
		return true // Assume ready if no status
	}

	conditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		return true // Assume ready if no conditions
	}

	for _, condition := range conditions {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(condMap, "type")
		condStatus, _, _ := unstructured.NestedString(condMap, "status")

		if condType == "Ready" && condStatus == "True" {
			return true
		}
	}

	return true // Default to ready for unknown resource types
}

// set the same labels used by helm install so that we can use the same
// approach to GC
func setTrackingLabelsAndAnnotations(obj *unstructured.Unstructured, cpName string) {
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

func propagateLabels(hook *v1alpha1.PostCreateHook, hcp *v1alpha1.ControlPlane, c client.Client) error {
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
		v, ok := hcpLabels[key]
		if !ok || ok && !(v == value) {
			updateRequired = true
		}
		hcpLabels[key] = value
	}
	hcp.SetLabels(hcpLabels)

	if updateRequired {
		if err := c.Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
