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
	"sync/atomic"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

// WaitForDeploymentReady returns once the identified Deployment is ready or gone
func WaitForDeploymentReady(kubeClient kubernetes.Interface, name, namespace string) error {
	return WaitForObjectState(context.Background(), kubeClient.AppsV1().RESTClient(),
		"deployments", &v1.Deployment{}, name, namespace,
		func(deploy *v1.Deployment) bool {
			return deploy.Status.ReadyReplicas == deploy.Status.Replicas &&
				deploy.Status.Replicas == *deploy.Spec.Replicas
		},
		true)
}

// WaitForStatefulSetReady returns once the identified StatefulSet is ready or gone
func WaitForStatefulSetReady(kubeClient kubernetes.Interface, name, namespace string) error {
	return WaitForObjectState(context.Background(), kubeClient.AppsV1().RESTClient(),
		"statefulsets", &v1.StatefulSet{}, name, namespace,
		func(stset *v1.StatefulSet) bool {
			return stset.Status.ReadyReplicas == stset.Status.Replicas &&
				stset.Status.Replicas == *stset.Spec.Replicas
		},
		true)
}

// WaitForNamespaceDeletion returns once the identified namespace is gone
func WaitForNamespaceDeletion(kubeClient kubernetes.Interface, name string) error {
	return WaitForObjectState(context.Background(), kubeClient.CoreV1().RESTClient(),
		"namespaces", &corev1.Namespace{},
		name, corev1.NamespaceAll,
		func(*corev1.Namespace) bool { return false },
		false)
}

// WaitForObjectState returns once the identified object meets the given test or
// the given context's Done() is closed.
// Returns nil in the first case, `ctx.Err()` in the last.
// Iff `!mustExist` then absence of the object is considered to meet the test.
func WaitForObjectState[T k8sruntime.Object](
	ctx context.Context,
	restClient k8srest.Interface,
	resource string, example T,
	name, namespace string,
	testState func(T) bool,
	mustExist bool,
) error {
	var success atomic.Bool
	cancelable, cancel := context.WithCancel(ctx)
	defer cancel()
	listwatch := cache.NewListWatchFromClient(
		restClient,
		resource,
		namespace,
		fields.OneTermEqualSelector("metadata.name", name),
	)
	consider := func(obj any) {
		typed := obj.(T)
		if testState(typed) {
			success.Store(true)
			cancel()
		}
	}
	store, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: listwatch,
		ObjectType:    example,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    consider,
			UpdateFunc: func(oldObj, newObj any) { consider(newObj) },
			DeleteFunc: func(obj any) {
				if !mustExist {
					success.Store(true)
					cancel()
				}
			},
		},
		ResyncPeriod: 0,
	})
	go controller.Run(cancelable.Done())
	if cache.WaitForCacheSync(cancelable.Done(), controller.HasSynced) {
		if len(store.ListKeys()) == 0 && !mustExist {
			return nil
		}
		<-cancelable.Done()
	}
	if success.Load() {
		return nil
	}
	// ctx.Done() must have been closed
	return ctx.Err()
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
