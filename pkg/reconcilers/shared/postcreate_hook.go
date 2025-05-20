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
    
    // Collect all hooks to process (legacy + new)
    var hooks []v1alpha1.PostCreateHookUse
    
    // Add legacy hook if specified
    if hcp.Spec.PostCreateHook != nil {
        hooks = append(hooks, v1alpha1.PostCreateHookUse{
            HookName: hcp.Spec.PostCreateHook,
            Vars:     hcp.Spec.PostCreateHookVars,
        })
    }
    
    // Add new hooks
    hooks = append(hooks, hcp.Spec.PostCreateHooks...)
    
    for _, hook := range hooks {
        if hook.HookName == nil {
            logger.Info("Skipping hook with nil name")
            continue
        }
        hookName := *hook.HookName
        
        // Skip already applied hooks
        if hcp.Status.PostCreateHooks != nil && hcp.Status.PostCreateHooks[hookName] {
            continue
        }
        
        logger.Info("Processing post-create hook", "hook", hookName)
        
        // Get hook definition
        pch := &v1alpha1.PostCreateHook{}
        if err := r.Client.Get(ctx, client.ObjectKey{Name: hookName}, pch); err != nil {
            return fmt.Errorf("failed to get PostCreateHook %s: %w", hookName, err)
        }
        
        // Build variables with precedence: defaults -> user vars -> system vars
        vars := make(map[string]interface{})
        
        // 1. Default vars from hook spec
        for _, dv := range pch.Spec.DefaultVars {
            vars[dv.Name] = dv.Value
        }
        
        // 2. User-provided vars from control plane
        for key, val := range hook.Vars {
            vars[key] = val
        }
        
        // 3. System variables
        vars["Namespace"] = namespace
        vars["ControlPlaneName"] = hcp.Name
        vars["HookName"] = hookName
        
        // Apply hook templates
        if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, pch, vars, hcp); err != nil {
            if util.IsTransientError(err) {
                return fmt.Errorf("transient error applying hook %s: %w", hookName, err)
            }
            logger.Error(err, "Failed to apply post-create hook", "hook", hookName)
            continue // Continue with other hooks
        }
        
        // Update status
        if hcp.Status.PostCreateHooks == nil {
            hcp.Status.PostCreateHooks = make(map[string]bool)
        }
        hcp.Status.PostCreateHooks[hookName] = true
        if err := r.Client.Status().Update(ctx, hcp); err != nil {
            return fmt.Errorf("failed to update status for hook %s: %w", hookName, err)
        }
        
        // Propagate labels
        if err := propagateLabels(pch, hcp, r.Client); err != nil {
            logger.Error(err, "Failed to propagate labels from hook", "hook", hookName)
        }
    }
    
    return nil
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
