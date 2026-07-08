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
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

func TestWaitForDeploymentReady_IsReady(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-deploy",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](3),
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	})

	err := WaitForDeploymentReady(t.Context(), fakeClient, "test-deploy", "test-ns")
	if err != nil {
		t.Fatalf("Failed to wait for ready Deployment, er=%v", err)
	}
}

func TestWaitForStatefulSetReady_IsReady(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-ss",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To[int32](3),
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	})

	err := WaitForStatefulSetReady(t.Context(), fakeClient, "test-ss", "test-ns")
	if err != nil {
		t.Fatalf("Failed to wait for ready StatefulSet, err=%v", err)
	}
}

func TestWaitForStatefulSetReady_StuckUnready(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      "test-ss",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: ptr.To[int32](3),
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      3,
			ReadyReplicas: 2,
		},
	})
	limited, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	err := WaitForStatefulSetReady(limited, fakeClient, "test-ss", "test-ns")
	switch {
	case err == nil:
		t.Fatal("Expected a timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected a timeout but got: %#v", err)
	}
}

func TestWaitForStatefulSetReady_Absent(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	limited, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	err := WaitForStatefulSetReady(limited, fakeClient, "test-ss", "test-ns")
	switch {
	case err == nil:
		t.Fatal("Expected a timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected a timeout but got: %#v", err)
	}
}

// TestWaitForNamespaceDeletion_Absent verifies a happy path:
// the namespace is absent from the start.
func TestWaitForNamespaceDeletion_Absent(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	err := WaitForNamespaceDeletion(t.Context(), fakeClient, "test-ns")
	if err != nil {
		t.Fatalf("expected nil when namespace is deleted, got: %v", err)
	}
}

func TestWaitForNamespaceDeletion_Stuck(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(&corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
		},
	})
	limited, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	err := WaitForNamespaceDeletion(limited, fakeClient, "test-ns")
	switch {
	case err == nil:
		t.Fatal("Expected a timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected a timeout but got: %#v", err)
	}
}
