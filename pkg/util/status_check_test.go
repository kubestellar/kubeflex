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
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func int32Ptr(i int32) *int32 { return &i }

// TestWaitForDeploymentReady_ChannelClosedBeforeReady verifies that closing the
// watch channel before the deployment reaches readiness returns an error instead
// of nil (false positive success).
func TestWaitForDeploymentReady_ChannelClosedBeforeReady(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fw := watch.NewFake()

	fakeClient.PrependWatchReactor("deployments", func(action k8stesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})

	// Close the channel immediately — simulates API server dropping the connection
	// before the deployment becomes ready.
	go fw.Stop()

	err := WaitForDeploymentReady(fakeClient, "test-deploy", "test-ns")
	if err == nil {
		t.Fatal("expected error when watch channel closes before deployment is ready, got nil")
	}
}

// TestWaitForDeploymentReady_ReadyBeforeChannelClose verifies the happy path:
// deployment reports ready replicas and the function returns nil.
func TestWaitForDeploymentReady_ReadyBeforeChannelClose(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fw := watch.NewFake()

	fakeClient.PrependWatchReactor("deployments", func(action k8stesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})

	deploy := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "test-ns"},
		Spec:       v1.DeploymentSpec{Replicas: int32Ptr(1)},
		Status: v1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
	}

	go fw.Modify(deploy)

	err := WaitForDeploymentReady(fakeClient, "test-deploy", "test-ns")
	if err != nil {
		t.Fatalf("expected nil when deployment is ready, got: %v", err)
	}
}

// TestWaitForStatefulSetReady_ChannelClosedBeforeReady verifies that closing the
// watch channel before the statefulset is ready returns an error.
func TestWaitForStatefulSetReady_ChannelClosedBeforeReady(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fw := watch.NewFake()

	fakeClient.PrependWatchReactor("statefulsets", func(action k8stesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})

	go fw.Stop()

	err := WaitForStatefulSetReady(fakeClient, "test-sts", "test-ns")
	if err == nil {
		t.Fatal("expected error when watch channel closes before statefulset is ready, got nil")
	}
}

// TestWaitForNamespaceDeletion_ChannelClosedBeforeDeletion verifies that closing
// the watch channel before the namespace is actually deleted returns an error.
func TestWaitForNamespaceDeletion_ChannelClosedBeforeDeletion(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fw := watch.NewFake()

	fakeClient.PrependWatchReactor("namespaces", func(action k8stesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})

	go fw.Stop()

	err := WaitForNamespaceDeletion(fakeClient, "test-ns")
	if err == nil {
		t.Fatal("expected error when watch channel closes before namespace is deleted, got nil")
	}
}

// TestWaitForNamespaceDeletion_DeletedSuccessfully verifies the happy path:
// the namespace is deleted and the function returns nil.
func TestWaitForNamespaceDeletion_DeletedSuccessfully(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fw := watch.NewFake()

	fakeClient.PrependWatchReactor("namespaces", func(action k8stesting.Action) (bool, watch.Interface, error) {
		return true, fw, nil
	})

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
	}

	go func() {
		fw.Modify(ns)
		fw.Delete(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
			TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
		})
	}()

	err := WaitForNamespaceDeletion(fakeClient, "test-ns")
	if err != nil {
		t.Fatalf("expected nil when namespace is deleted, got: %v", err)
	}
}
