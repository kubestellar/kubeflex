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
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	vars := Vars{
		Namespace:        namespace,
		ControlPlaneName: hcp.Name,
		HookName:         *hcp.Spec.PostCreateHook,
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
		return errors.Join(fmt.Errorf("error retrieving post create hook %s", *hcp.Spec.PostCreateHook), err)
	}

	if err := applyPostCreateHook(ctx, r.ClientSet, r.DynamicClient, hook, vars); err != nil {
		return err
	}

	return nil
}

func applyPostCreateHook(ctx context.Context, clientSet *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, hook *v1alpha1.PostCreateHook, vars Vars) error {
	logger := clog.FromContext(ctx)
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
			return errors.New("null object in template")
		}

		gvk := util.GetGroupVersionKindFromObject(obj)
		gvr, err := util.GroupVersionKindToResource(clientSet, gvk)
		if err != nil {
			return err
		}

		clusterScoped, err := util.IsClusterScoped(gvk, apiResourceLists)
		if err != nil {
			return err
		}

		logger.Info("Applying", "object", util.GenerateObjectInfoString(*obj))

		if clusterScoped {
			_, err = dynamicClient.Resource(*gvr).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		} else {
			_, err = dynamicClient.Resource(*gvr).Namespace(vars.Namespace).Apply(context.TODO(), obj.GetName(), obj, metav1.ApplyOptions{FieldManager: FieldManager})
		}
		if err != nil {
			return err
		}
	}
	return nil
}
