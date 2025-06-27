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
			HookName: hookName,
			Vars:     hcp.Spec.PostCreateHookVars,
		})
		seen[hookName] = true
	}
	
	// Add new hooks in declared order, skipping duplicates
	for _, hook := range hcp.Spec.PostCreateHooks {
		if hook.HookName != "" {
			hookName := hook.HookName
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
		hookName := hook.HookName
		
		// Check if hook is already applied
		_, ok := hcp.Status.PostCreateHooks[hookName]
		if ok {
			// Hook is applied, check if resources are ready only if WaitForPostCreateHooks is enabled
			if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
				logger.Info("Checking post-create hook resources", "hook", hookName)
				if err := r.checkPostCreateHookResources(ctx, hcp, hookName, hook.Vars); err != nil {
					errs = append(errs, fmt.Errorf("error checking post create hook resources for %s: %w", hookName, err))
					allHooksApplied = false
				}
			}
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
		if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, pch, vars, hcp); err != nil {
			if util.IsTransientError(err) {
				errs = append(errs, fmt.Errorf("transient error applying hook %s: %w", hookName, err))
			} else {
				logger.Error(err, "Permanent error applying post-create hook", "hook", hookName)
				errs = append(errs, fmt.Errorf("permanent error applying hook %s: %w", hookName, err))
			}
			allHooksApplied = false
			continue
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
	
	// Update PostCreateHookCompleted status based on all hooks and their resources
	if hcp.Spec.WaitForPostCreateHooks != nil && *hcp.Spec.WaitForPostCreateHooks {
		allResourcesReady := true
		for _, hook := range hooks {
			hookName := hook.HookName
			if hcp.Status.PostCreateHooks[hookName] {
				if err := r.checkPostCreateHookResources(ctx, hcp, hookName, hook.Vars); err != nil {
					logger.Info("Hook resources not ready", "hook", hookName, "error", err)
					allResourcesReady = false
				}
			} else {
				allResourcesReady = false
			}
		}
		hcp.Status.PostCreateHookCompleted = allResourcesReady && allHooksApplied
	} else {
		// If not waiting for resources, just check if all hooks are applied
		hcp.Status.PostCreateHookCompleted = allHooksApplied
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

// checkPostCreateHookResources checks if all resources created by a specific post create hook are ready
func (r *BaseReconciler) checkPostCreateHookResources(ctx context.Context, hcp *v1alpha1.ControlPlane, hookName string, hookVars map[string]string) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Get the post create hook
	hook := &v1alpha1.PostCreateHook{
		ObjectMeta: metav1.ObjectMeta{
			Name: hookName,
		},
	}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(hook), hook, &client.GetOptions{}); err != nil {
		return fmt.Errorf("error retrieving post create hook %s: %w", hookName, err)
	}

	// Build variables with proper precedence
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
		// Render the template with variables first
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

		// Check resource status based on type
		ready, err := r.checkResourceStatus(ctx, gvr, obj.GetName(), namespace, gvk.Kind)
		if err != nil {
			return fmt.Errorf("error checking resource status: %w", err)
		}

		if !ready {
			logger.Info("Resource not ready", "kind", gvk.Kind, "name", obj.GetName(), "hook", hookName)
			return fmt.Errorf("resource %s/%s not ready", gvk.Kind, obj.GetName())
		}
	}

	return nil
}

// checkResourceStatus checks if a specific resource is ready based on its type
func (r *BaseReconciler) checkResourceStatus(ctx context.Context, gvr schema.GroupVersionResource, name, namespace, kind string) (bool, error) {
	switch kind {
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
		logger := clog.FromContext(ctx)
		logger.Info("Job status debug", "name", name, "succeeded", job.Status.Succeeded, "conditions", job.Status.Conditions)
		return job.Status.Succeeded > 0, nil

	case "DaemonSet":
		daemonset := &appsv1.DaemonSet{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, daemonset); err != nil {
			return false, err
		}
		return daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled, nil

	default:
		// For other resource types, consider them ready if they exist
		_, err := r.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		return err == nil, nil
	}
}

func applyPostCreateHook(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, hook *v1alpha1.PostCreateHook, vars map[string]interface{}, hcp *v1alpha1.ControlPlane) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)
	apiResourceLists, err := clientSet.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return err
	}
	for _, template := range hook.Spec.Templates {
		raw := template.Raw
		rendered, err := util.RenderYAML(raw, vars)
		if err != nil {
			return err
		}

		obj, err := util.ToUnstructured(rendered)
		if err != nil {
			return err
		}

		if obj == nil {
			return fmt.Errorf("null object in template")
		}

		gvk := util.GetGroupVersionKindFromObject(obj)
		gvr, err := util.GVKToGVR(clientSet, gvk)
		if err != nil {
			return err
		}

		clusterScoped, err := util.IsClusterScoped(gvk, apiResourceLists)
		if err != nil {
			return err
		}

		logger.Info("Applying", "object", util.GenerateObjectInfoString(*obj), "cpNamespace", namespace)

		if clusterScoped {
			setTrackingLabelsAndAnnotations(obj, hcp.Name)
			_, err = dynamicClient.Resource(gvr).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		}
		if err != nil {
			return err
		}
	}
	return nil
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
		if err := c.Update(context.Background(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
