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

	if hcp.Spec.PostCreateHook == nil {
		return nil
	}

	// Hook is only applied once after CP creation
	_, ok := hcp.Status.PostCreateHooks[*hcp.Spec.PostCreateHook]
	if ok {
		// Check if all resources are ready
		if err := r.checkPostCreateHookResources(ctx, hcp); err != nil {
			return fmt.Errorf("error checking post create hook resources: %w", err)
		}
		return nil
	}

	logger.Info("Running ReconcileUpdatePostCreateHook", "post-create-hook", *hcp.Spec.PostCreateHook)

	// Get the post create hook
	hook := &v1alpha1.PostCreateHook{
		ObjectMeta: metav1.ObjectMeta{
			Name: *hcp.Spec.PostCreateHook,
		},
	}
	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(hook), hook, &client.GetOptions{}); err != nil {
		return fmt.Errorf("error retrieving post create hook %s: %w", *hcp.Spec.PostCreateHook, err)
	}

	// Build variables with proper precedence:
	// 1. Default vars from PostCreateHook
	// 2. User-provided vars from ControlPlane (override defaults)
	// 3. Built-in system vars (highest priority)
	vars := make(map[string]interface{})

	// Add default variables from hook spec
	for _, dv := range hook.Spec.DefaultVars {
		vars[dv.Name] = dv.Value
	}

	// Override with user-provided variables from control plane
	for key, val := range hcp.Spec.PostCreateHookVars {
		vars[key] = val
	}

	// Add system variables (highest priority)
	vars["Namespace"] = namespace
	vars["ControlPlaneName"] = hcp.Name
	vars["HookName"] = *hcp.Spec.PostCreateHook

	// Apply the hook templates
	if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, hook, vars, hcp); err != nil {
		if util.IsTransientError(err) {
			return fmt.Errorf("failed to apply post-create hook: %w", err) // Retry
		}
		logger.Error(err, "Failed to apply post-create hook", "hook", *hcp.Spec.PostCreateHook)
	}

	// Update status if hook was successfully applied
	if hcp.Status.PostCreateHooks == nil {
		hcp.Status.PostCreateHooks = make(map[string]bool)
	}
	hcp.Status.PostCreateHooks[*hcp.Spec.PostCreateHook] = true

	if err := r.Client.Status().Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update control plane status: %w", err)
	}

	// Propagate labels from hook to control plane
	if err := propagateLabels(hook, hcp, r.Client); err != nil {
		return fmt.Errorf("failed to propagate labels: %w", err)
	}

	return nil
}

// checkPostCreateHookResources checks if all resources created by the post create hook are ready
func (r *BaseReconciler) checkPostCreateHookResources(ctx context.Context, hcp *v1alpha1.ControlPlane) error {
	logger := clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// Get the post create hook
	hook := &v1alpha1.PostCreateHook{
		ObjectMeta: metav1.ObjectMeta{
			Name: *hcp.Spec.PostCreateHook,
		},
	}
	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(hook), hook, &client.GetOptions{}); err != nil {
		return fmt.Errorf("error retrieving post create hook %s: %w", *hcp.Spec.PostCreateHook, err)
	}

	// Check each template's resource status
	for _, template := range hook.Spec.Templates {
		obj, err := util.ToUnstructured(template.Raw)
		if err != nil {
			return fmt.Errorf("error converting template to unstructured: %w", err)
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
			logger.Info("Resource not ready", "kind", gvk.Kind, "name", obj.GetName())
			hcp.Status.PostCreateHookCompleted = false
			return r.Client.Status().Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{})
		}
	}

	// All resources are ready
	hcp.Status.PostCreateHookCompleted = true
	return r.Client.Status().Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{})
}

// checkResourceStatus checks if a specific resource is ready based on its type
func (r *BaseReconciler) checkResourceStatus(ctx context.Context, gvr schema.GroupVersionResource, name, namespace, kind string) (bool, error) {
	switch kind {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment); err != nil {
			return false, err
		}
		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas, nil

	case "StatefulSet":
		statefulset := &appsv1.StatefulSet{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, statefulset); err != nil {
			return false, err
		}
		return statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas, nil

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
			_, err = dynamicClient.Resource(gvr).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
			_, err = dynamicClient.Resource(gvr).Namespace(namespace).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
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
		if err := c.Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
