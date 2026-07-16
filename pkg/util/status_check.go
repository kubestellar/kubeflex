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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	informersappsv1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

// WaitForDeploymentReady returns once the identified Deployment exists and is ready
// or 10 minutes pass, returning `context.DeadlineExceeded` in the latter case.
func WaitForDeploymentReady(ctx context.Context, kubeClient kubernetes.Interface, name, namespace string) error {
	inf := informersappsv1.NewFilteredDeploymentInformer(kubeClient, namespace, 0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, OptionsAddName(name))
	err := WaitForObjectState(ctx, 10*time.Minute,
		inf, nil,
		func(deploy *appsv1.Deployment) bool {
			return deploy.Status.ReadyReplicas == deploy.Status.Replicas &&
				deploy.Status.Replicas == *deploy.Spec.Replicas
		},
		true)
	if err != nil {
		return fmt.Errorf("the Deployment %s/%s did not become ready in time: %w", namespace, name, err)
	}
	return nil
}

// WaitForStatefulSetReady returns once the identified StatefulSet exists and is ready
// or 10 minutes pass, returning `context.DeadlineExceeded` in the latter case.
func WaitForStatefulSetReady(ctx context.Context, kubeClient kubernetes.Interface, name, namespace string) error {
	inf := informersappsv1.NewFilteredStatefulSetInformer(kubeClient, namespace, 0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, OptionsAddName(name))
	err := WaitForObjectState(ctx, 10*time.Minute,
		inf, nil,
		func(stset *appsv1.StatefulSet) bool {
			return stset.Status.ReadyReplicas == stset.Status.Replicas &&
				stset.Status.Replicas == *stset.Spec.Replicas
		},
		true)
	if err != nil {
		return fmt.Errorf("the StatefulSet %s/%s did not become ready in time: %w", namespace, name, err)
	}
	return nil
}

// WaitForNamespaceDeletion returns once the identified namespace is gone
// or 10 minutes pass, returning a `context.DeadlineExceeded` in the latter case.
func WaitForNamespaceDeletion(ctx context.Context, kubeClient kubernetes.Interface, name string) error {
	li := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0,
		informers.WithTweakListOptions(OptionsAddName(name))).Core().V1().Namespaces()
	err := WaitForObjectState(ctx, 10*time.Minute,
		li.Informer(), li.Lister().List,
		func(*corev1.Namespace) bool { return false },
		false)
	if err != nil {
		return fmt.Errorf("namespace %s was not deleted in time: %w", name, err)
	}
	return nil
}

func OptionsAddName(name string) func(lo *metav1.ListOptions) {
	return func(lo *metav1.ListOptions) {
		newSel := fields.OneTermEqualSelector("metadata.name", name).String()
		if lo.FieldSelector == "" {
			lo.FieldSelector = newSel
		} else {
			lo.FieldSelector = lo.FieldSelector + "," + newSel
		}
	}
}

// WaitForObjectState waits for one Kubernetes API object to reach a state
// that passes the given test or the timeout to pass or the context's `Done()` to be closed.
// Returns nil in the first case, `context.DeadlineExceeded` in the second, `ctx.Err()` in the last.
// The informer and lister must be focused on just the object in question.
// Iff `!mustExist` then absence of the object is considered to meet the test.
// If `mustExist` then the lister may be `nil`.
func WaitForObjectState[T k8sruntime.Object](
	ctx context.Context,
	timeout time.Duration,
	informer cache.SharedIndexInformer,
	lister func(labels.Selector) ([]T, error),
	testState func(T) bool,
	mustExist bool,
) error {
	var success atomic.Bool
	cancelable, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	consider := func(obj any) {
		typed := obj.(T)
		if testState(typed) {
			success.Store(true)
			cancel()
		}
	}
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    consider,
			UpdateFunc: func(oldObj, newObj any) { consider(newObj) },
			DeleteFunc: func(obj any) {
				if !mustExist {
					success.Store(true)
					cancel()
				}
			},
		})

	go informer.Run(cancelable.Done())
	if cache.WaitForCacheSync(cancelable.Done(), informer.HasSynced) {
		if !mustExist {
			list, err := lister(labels.Everything())
			if err != nil { // Only happens if wrong kind of objects are in the cache
				return fmt.Errorf("impossible failure in lister: %w", err)
			}
			if len(list) == 0 {
				return nil
			}
		}
		<-cancelable.Done()
	}
	if success.Load() {
		return nil
	}
	// either the timeout happened or ctx.Done() was closed
	return cancelable.Err()
}

// IsAPIServiceAvail tests whether the API service of the given ControlPlane is reachable and working.
// If not, an explanatory message is logged to the context's logger.
func IsAPIServiceAvail(ctx context.Context, kubeClient kubernetes.Interface, hcp *tenancyv1alpha1.ControlPlane) bool {
	switch hcp.Spec.Type {
	case tenancyv1alpha1.ControlPlaneTypeExternal, tenancyv1alpha1.ControlPlaneTypeHost:
		return true
	}
	logger := clog.FromContext(ctx)
	secretRef := hcp.Status.SecretRef
	if secretRef == nil {
		logger.V(3).Info("ControlPlane's API service is not reachable because status.secretRef is nil")
		return false
	}
	if secretRef.Namespace == "" || secretRef.Name == "" {
		logger.V(3).Info("ControlPlane's API service is not reachable because status.secretRef has an empty Secret name or namespace")
		return false
	}
	if secretRef.InClusterKey == "" {
		logger.V(3).Info("ControlPlane's API service is not reachable because status.secretRef.inClusterKey is empty")
		return false
	}
	secret, err := kubeClient.CoreV1().Secrets(secretRef.Namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
	if err != nil {
		logger.V(3).Info("ControlPlane's API service is not reachable because Get of Secret fails", "secretName", secretRef.Name)
		return false
	}
	cpKubeconfigBytes := secret.Data[secretRef.InClusterKey]
	if len(cpKubeconfigBytes) == 0 {
		logger.V(3).Info("ControlPlane's API service is not reachable because Secret has empty value", "secretName", secretRef.Name, "key", secretRef.InClusterKey)
		return false
	}
	cpKubeconfig, err := clientcmd.Load(cpKubeconfigBytes)
	if err != nil {
		logger.V(3).Info("ControlPlane's API service is not reachable because loading of its kubeconfig failed", "err", err)
		return false
	}
	cpConfig, err := clientcmd.NewDefaultClientConfig(*cpKubeconfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		logger.V(3).Info("ControlPlane's API service is not reachable because its kubeconfig fails conversion to rest.Config", "err", err)
		return false
	}
	if cpConfig.Timeout == 0 || cpConfig.Timeout > 25*time.Second {
		cpConfig.Timeout = 5 * time.Second
	}
	cpClient, err := kubernetes.NewForConfig(cpConfig)
	if err != nil {
		logger.V(3).Info("ControlPlane's API service is not reachable because its kubeconfig fails conversion to kubernetes.Clientset", "err", err)
		return false
	}
	coreGroupVersion := corev1.SchemeGroupVersion.String()
	_, err = cpClient.ServerResourcesForGroupVersion(coreGroupVersion)
	if err != nil {
		logger.V(3).Info("ControlPlane's API service fails to list resources in corev1", "err", err)
		return false
	}
	return true
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
		s := &appsv1.StatefulSet{
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
		d := &appsv1.Deployment{
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
		d := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetAPIServerDeploymentNameByControlPlaneType(string(hcp.Spec.Type)),
				Namespace: ns.Name,
			},
		}
		err := c.Get(context.Background(), types.NamespacedName{Name: d.Name, Namespace: d.Namespace}, d)
		return err == nil, nil // Just check if it exists, not if it's ready
	case tenancyv1alpha1.ControlPlaneTypeVCluster:
		s := &appsv1.StatefulSet{
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
