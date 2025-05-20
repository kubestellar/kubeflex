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
    batchv1 "k8s.io/api/batch/v1"
    "k8s.io/apimachinery/pkg/util/wait"
    "time"
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
    
    // Handle legacy hook format
    if hcp.Spec.PostCreateHook != nil {
        logger.Info("Migrating legacy PostCreateHook to new format")
        
        // Preserve existing status if present
        existingStatus := make(map[string]bool)
        if hcp.Status.PostCreateHooks != nil {
            existingStatus = hcp.Status.PostCreateHooks
        }

        // Create new hooks array
        hcp.Spec.PostCreateHooks = append(hcp.Spec.PostCreateHooks, v1alpha1.PostCreateHookUse{
            HookName: hcp.Spec.PostCreateHook,
            Vars:     hcp.Spec.PostCreateHookVars,
        })

        // Clear legacy fields
        hcp.Spec.PostCreateHook = nil
        hcp.Spec.PostCreateHookVars = nil
        
        // Update the spec first
        if err := r.Client.Update(ctx, hcp); err != nil {
            return fmt.Errorf("failed to migrate legacy hook: %w", err)
        }

        // Restore status
        hcp.Status.PostCreateHooks = existingStatus
    }

    // Process all hooks
    for _, hookUse := range hcp.Spec.PostCreateHooks {
        if hookUse.HookName == nil {
            continue
        }

        hookName := *hookUse.HookName
        if hcp.Status.PostCreateHooks != nil && hcp.Status.PostCreateHooks[hookName] {
            continue
        }

        logger.Info("Processing post-create hook", "hook", hookName)

        hook := &v1alpha1.PostCreateHook{
            ObjectMeta: metav1.ObjectMeta{
                Name: hookName,
            },
        }
        if err := r.Client.Get(ctx, client.ObjectKeyFromObject(hook), hook); err != nil {
            return fmt.Errorf("error retrieving post create hook %s: %w", hookName, err)
        }

        // Build variables with proper precedence
        vars := make(map[string]interface{})
        for _, dv := range hook.Spec.DefaultVars {
            vars[dv.Name] = dv.Value
        }
        for k, v := range hookUse.Vars {
            vars[k] = v
        }
        vars["Namespace"] = namespace
        vars["ControlPlaneName"] = hcp.Name
        vars["HookName"] = hookName

        if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, hook, vars, hcp); err != nil {
            logger.Error(err, "Failed to apply post-create hook", "hook", hookName)
            continue
        }

        if hcp.Status.PostCreateHooks == nil {
            hcp.Status.PostCreateHooks = make(map[string]bool)
        }
        hcp.Status.PostCreateHooks[hookName] = true
    }

    if err := r.Client.Status().Update(ctx, hcp); err != nil {
        return fmt.Errorf("failed to update control plane status: %w", err)
    }

    return nil
}

func createAndMonitorJobs(ctx context.Context, clientset *kubernetes.Clientset, hook *v1alpha1.PostCreateHook, vars map[string]interface{}, hcp *v1alpha1.ControlPlane) error {
    namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

    for _, jobTemplate := range hook.Spec.Jobs {
        // Render job template with variables
        job := &batchv1.Job{
            ObjectMeta: metav1.ObjectMeta{
                Name:      jobTemplate.ObjectMeta.Name,
                Namespace: namespace,
               Labels:    jobTemplate.ObjectMeta.Labels,
            },
            Spec: jobTemplate.Spec,
        }

        // Create the Job
        _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
        if err != nil {
            return fmt.Errorf("failed to create job %s: %w", job.Name, err)
        }

        // Monitor job completion
       err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 10*time.Minute, true, 
            func(ctx context.Context) (bool, error) {
                currentJob, err := clientset.BatchV1().Jobs(namespace).Get(ctx, job.Name, metav1.GetOptions{})
                if err != nil {
                    return false, err
               }
                
                if currentJob.Status.Succeeded > 0 {
                    return true, nil
               }
                if currentJob.Status.Failed > *currentJob.Spec.BackoffLimit {
                    return false, fmt.Errorf("job %s failed", job.Name)
                }
                return false, nil
            })

        if err != nil {
            return fmt.Errorf("job %s failed: %w", job.Name, err)
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
