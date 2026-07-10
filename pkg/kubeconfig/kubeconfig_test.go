package kubeconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Setup and teardown helpers
func setupTestKubeconfigWithContext(ctxName string) *api.Config {
	kconf := api.NewConfig()
	kconf.Contexts[ctxName] = &api.Context{
		Cluster:  "test-cluster",
		AuthInfo: "test-user",
	}
	return kconf
}

func teardownTestKubeconfig(kconf *api.Config, ctxName string) {
	delete(kconf.Contexts, ctxName)
}

func TestAssignControlPlaneToContext_Positive(t *testing.T) {
	ctxName := "test-context"
	cpName := "test-controlplane"
	kconf := setupTestKubeconfigWithContext(ctxName)
	defer teardownTestKubeconfig(kconf, ctxName)

	err := AssignControlPlaneToContext(kconf, cpName, ctxName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, ok := kconf.Contexts[ctxName].Extensions[ExtensionKubeflexKey]
	if !ok {
		t.Fatalf("expected kubeflex extension to be set in context")
	}

	kflexConfig, err := NewKubeflexContextConfig(*kconf, ctxName)
	if err != nil {
		t.Fatalf("expected no error creating kubeflex context config, got: %v", err)
	}
	runtimeExt := NewRuntimeKubeflexExtension()
	err = kflexConfig.ConvertExtensionsToRuntimeExtension(runtimeExt)
	if err != nil {
		t.Fatalf("expected no error converting extensions to runtime extension, got: %v", err)
	}

	if got := runtimeExt.Data[ExtensionControlPlaneName]; got != cpName {
		t.Errorf("expected controlplane-name to be '%s', got '%s'", cpName, got)
	}
}

func TestAssignControlPlaneToContext_Negative_ContextDoesNotExist(t *testing.T) {
	ctxName := "nonexistent-context"
	cpName := "test-controlplane"
	kconf := api.NewConfig()

	err := AssignControlPlaneToContext(kconf, cpName, ctxName)
	if err == nil {
		t.Fatalf("expected error for nonexistent context, got nil")
	}
}

func TestWatchForSecretCreation_Happy(t *testing.T) {
	fakeClient := k8s_fake.NewSimpleClientset(&corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-cp-system",
			Name:      "test-secret",
		},
		Type: corev1.SecretTypeOpaque,
	})

	err := WatchForSecretCreation(t.Context(), fakeClient, "test-cp", "test-secret")
	if err != nil {
		t.Fatalf("Failed with err=%#v", err)
	}
}

func TestWatchForSecretCreation_Absent(t *testing.T) {
	fakeClient := k8s_fake.NewSimpleClientset()
	limited, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	err := WatchForSecretCreation(limited, fakeClient, "test-cp", "test-secret")
	switch {
	case err == nil:
		t.Fatal("Expected timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected timeout but got: %#v", err)
	}
}

func TestWaitForNamespaceReady_Happy(t *testing.T) {
	ctx := t.Context()
	fakeClient := k8s_fake.NewSimpleClientset(
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cp-system",
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceActive,
			},
		},
	)

	err := WaitForNamespaceReady(ctx, fakeClient, "test-cp")
	switch {
	case err == nil:
	case errors.Is(err, context.DeadlineExceeded):
		t.Fatal("Expected success but got timeout")
	default:
		t.Fatalf("Expected timeout but got err=%#v", err)
	}
}

func TestWaitForNamespaceReady_StuckTerminating(t *testing.T) {
	ctx := t.Context()
	// The fake clientset does not apply field selectors in
	// list or watch operations, so we can not test
	// ignoring objects with the wrong name.
	fakeClient := k8s_fake.NewSimpleClientset(
		&corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cp-system",
			},
			Status: corev1.NamespaceStatus{
				Phase: corev1.NamespaceTerminating,
			},
		},
	)

	err := WaitForNamespaceReady(ctx, fakeClient, "test-cp")
	switch {
	case err == nil:
		t.Fatal("Expected timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected timeout but got err=%#v", err)
	}
}

func TestWaitForNamespaceReady_Absent(t *testing.T) {
	fakeClient := k8s_fake.NewSimpleClientset()
	limited, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	err := WaitForNamespaceReady(limited, fakeClient, "test-cp")
	switch {
	case err == nil:
		t.Fatal("Expected timeout but got success")
	case errors.Is(err, context.DeadlineExceeded):
	default:
		t.Fatalf("Expected timeout but got err=%#v", err)
	}
}
