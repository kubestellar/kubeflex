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

package util

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

const (
	ManagedByKey                      = "app.kubernetes.io/managed-by"
	HelmReleaseNamespaceAnnotationKey = "meta.helm.sh/release-namespace"
)

func SetClusterScopedOwnerRefs(c crc.Client, scheme *runtime.Scheme, hcp *tenancyv1alpha1.ControlPlane) error {
	namespace := GenerateNamespaceFromControlPlaneName(hcp.Name)

	matchingLabels := crc.MatchingLabels{ManagedByKey: "Helm"}

	if err := setClusterRolesOwner(c, scheme, hcp, namespace, matchingLabels); err != nil {
		return err
	}

	if err := setClusterRoleBindingsOwner(c, scheme, hcp, namespace, matchingLabels); err != nil {
		return err
	}

	if err := setCRDOwner(c, scheme, hcp, namespace, matchingLabels); err != nil {
		return err
	}

	return nil
}

func setClusterRolesOwner(c crc.Client, scheme *runtime.Scheme, hcp *tenancyv1alpha1.ControlPlane, namespace string, matchingLabels crc.MatchingLabels) error {
	clusterRoleList := &rbacv1.ClusterRoleList{}
	err := c.List(context.Background(), clusterRoleList, matchingLabels)
	if err != nil {
		return err
	}

	for _, clusterRole := range clusterRoleList.Items {
		ns, ok := clusterRole.Annotations[HelmReleaseNamespaceAnnotationKey]
		if ok && ns == namespace {
			if err := controllerutil.SetControllerReference(hcp, &clusterRole, scheme); err != nil {
				return err
			}
			err = c.Update(context.Background(), &clusterRole)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func setClusterRoleBindingsOwner(c crc.Client, scheme *runtime.Scheme, hcp *tenancyv1alpha1.ControlPlane, namespace string, matchingLabels crc.MatchingLabels) error {
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	err := c.List(context.Background(), clusterRoleBindingList, matchingLabels)
	if err != nil {
		return err
	}

	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		ns, ok := clusterRoleBinding.Annotations[HelmReleaseNamespaceAnnotationKey]
		if ok && ns == namespace {
			if err := controllerutil.SetControllerReference(hcp, &clusterRoleBinding, scheme); err != nil {
				return err
			}
			err = c.Update(context.Background(), &clusterRoleBinding)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func setCRDOwner(c crc.Client, scheme *runtime.Scheme, hcp *tenancyv1alpha1.ControlPlane, namespace string, matchingLabels crc.MatchingLabels) error {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	err := c.List(context.Background(), crdList, matchingLabels)
	if err != nil {
		return err
	}

	for _, crd := range crdList.Items {
		ns, ok := crd.Annotations[HelmReleaseNamespaceAnnotationKey]
		if ok && ns == namespace {
			if err := controllerutil.SetControllerReference(hcp, &crd, scheme); err != nil {
				return err
			}
			err = c.Update(context.Background(), &crd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
