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
	if hcp.Spec.PostCreateHook == nil {
		return nil
	}

	// hook is only applied once after CP creation
	_, ok := hcp.Status.PostCreateHooks[*hcp.Spec.PostCreateHook]
	if ok {
		return nil
	}

	// built-in vars
	vars := map[string]interface{}{
		"Namespace":        namespace,
		"ControlPlaneName": hcp.Name,
		"HookName":         *hcp.Spec.PostCreateHook,
	}

	// user-defined vars
	for key, val := range hcp.Spec.PostCreateHookVars {
		vars[key] = val
	}

	logger.Info("Running ReconcileUpdatePostCreateHook", "post-create-hook", *hcp.Spec.PostCreateHook)

	// get the post create hook
	hook := &v1alpha1.PostCreateHook{
		ObjectMeta: metav1.ObjectMeta{
			Name: *hcp.Spec.PostCreateHook,
		},
	}
	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(hook), hook, &client.GetOptions{})
	if err != nil {
		return fmt.Errorf("error retrieving post create hook %s %s", *hcp.Spec.PostCreateHook, err)
	}

	if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, hook, vars, hcp); err != nil {
		return err
	}

	// if hook was successfully applied update status
	if hcp.Status.PostCreateHooks == nil {
		hcp.Status.PostCreateHooks = map[string]bool{}
	}
	hcp.Status.PostCreateHooks[*hcp.Spec.PostCreateHook] = true
	if err := r.Client.Status().Update(context.TODO(), hcp, &client.SubResourceUpdateOptions{}); err != nil {
		return err
	}

	if err := propagateLabels(hook, hcp, r.Client); err != nil {
		return err
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

		logger.Info("Applying", "object", util.GenerateObjectInfoString(*obj))

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
	if hookLabels == nil || hookLabels != nil && len(hookLabels) == 0 {
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
