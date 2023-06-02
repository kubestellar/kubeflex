package util

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

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
