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
	"fmt"
	"log"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

// TODO - refactor in a single base "WaitFor" function that can operate on the resource types
// needed here

func WaitForDeploymentReady(clientset kubernetes.Clientset, name, namespace string) error {
	watcher, err := clientset.AppsV1().Deployments(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector:   fmt.Sprintf("metadata.name=%s", name),
		ResourceVersion: "0",
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		event, ok := <-watcher.ResultChan()
		if !ok {
			return nil
		}

		switch event.Type {
		case watch.Error:
			log.Println("Error watching deployment:", watch.Error)
		case watch.Added, watch.Modified:
			deploy := event.Object.(*v1.Deployment)
			if deploy.Status.ReadyReplicas == deploy.Status.Replicas && deploy.Status.Replicas == *deploy.Spec.Replicas {
				return nil
			}
		case watch.Deleted:
			return nil
		}
	}
}

func WaitForStatefulSetReady(clientset kubernetes.Clientset, name, namespace string) error {
	watcher, err := clientset.AppsV1().StatefulSets(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		event, ok := <-watcher.ResultChan()
		if !ok {
			return nil
		}

		switch event.Type {
		case watch.Error:
			log.Println("Error watching statefulset:", watch.Error)
		case watch.Added, watch.Modified:
			stset := event.Object.(*v1.StatefulSet)
			if stset.Status.ReadyReplicas == stset.Status.Replicas && stset.Status.Replicas == *stset.Spec.Replicas {
				return nil
			}
		case watch.Deleted:
			return nil
		}
	}
}

func WaitForNamespaceDeletion(clientset kubernetes.Clientset, name string) error {
	watcher, err := clientset.CoreV1().Namespaces().Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		event, ok := <-watcher.ResultChan()
		if !ok {
			return nil
		}

		switch event.Type {
		case watch.Error:
			log.Println("Error watching namespace:", watch.Error)
		case watch.Added, watch.Modified:
			namespace := event.Object.(*corev1.Namespace)
			if namespace.Status.Phase == corev1.NamespaceTerminating {
				continue
			}
		case watch.Deleted:
			return nil
		}
	}
}

func IsAPIServerDeploymentReady(log logr.Logger, c client.Client, hcp tenancyv1alpha1.ControlPlane) (bool, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateNamespaceFromControlPlaneName(hcp.Name),
		},
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: ns.Name}, ns); err != nil {
		return false, err
	}

	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeHost, tenancyv1alpha1.ControlPlaneTypeExternal:
		// host or external is always available
		return true, nil
	case tenancyv1alpha1.ControlPlaneTypeVCluster, tenancyv1alpha1.ControlPlaneTypeK3s:
		s := &v1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetAPIServerDeploymentNameByControlPlaneType(string(hcp.Spec.Type)),
				Namespace: ns.Name,
			},
		}

		if err := c.Get(context.Background(), types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, s); err != nil {
			return false, err
		}

		// we need to ensure that there is al least one replica in the spec
		return s.Status.ReadyReplicas == s.Status.Replicas &&
			s.Status.Replicas == *s.Spec.Replicas &&
			*s.Spec.Replicas > 0, nil
	case tenancyv1alpha1.ControlPlaneTypeK8S, tenancyv1alpha1.ControlPlaneTypeOCM:
		d := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetAPIServerDeploymentNameByControlPlaneType(string(hcp.Spec.Type)),
				Namespace: ns.Name,
			},
		}

		if err := c.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, d); err != nil {
			log.Error(err, "Failed to get deployment", "name", d.Name, "namespace", d.Namespace)
			return false, err
		}

		log.Info("Deployment status check", "name", d.Name, "namespace", d.Namespace, 
			"readyReplicas", d.Status.ReadyReplicas, "replicas", d.Status.Replicas, 
			"specReplicas", *d.Spec.Replicas)

		// we need to ensure that there is al least one replica in the spec
		return d.Status.ReadyReplicas == d.Status.Replicas &&
			d.Status.Replicas == *d.Spec.Replicas &&
			*d.Spec.Replicas > 0, nil
	default:
		log.Error(fmt.Errorf("control plane type not supported"), "isAPIServerDeploymentReady failed", "type", hcp.Spec.Type)
		return false, nil
	}
}

func IsAPIServerDeploymentExists(c client.Client, hcp tenancyv1alpha1.ControlPlane) (bool, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateNamespaceFromControlPlaneName(hcp.Name),
		},
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: ns.Name}, ns); err != nil {
		return false, err
	}

	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeHost, tenancyv1alpha1.ControlPlaneTypeExternal:
		return true, nil
	case tenancyv1alpha1.ControlPlaneTypeK8S, tenancyv1alpha1.ControlPlaneTypeOCM:
		d := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetAPIServerDeploymentNameByControlPlaneType(string(hcp.Spec.Type)),
				Namespace: ns.Name,
			},
		}
		err := c.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, d)
		return err == nil, nil // Just check if it exists, not if it's ready
	case tenancyv1alpha1.ControlPlaneTypeVCluster:
		s := &v1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetAPIServerDeploymentNameByControlPlaneType(string(hcp.Spec.Type)),
				Namespace: ns.Name,
			},
		}
		err := c.Get(context.Background(), types.NamespacedName{Name: s.Name, Namespace: s.Namespace}, s)
		return err == nil, nil // Just check if it exists, not if it's ready
	}
	return false, nil
}
