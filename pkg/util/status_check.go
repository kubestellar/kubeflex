package util

import (
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func WaitForDeploymentReady(clientset kubernetes.Clientset, name, namespace string) error {
	fmt.Printf("Waiting for kube api server to become ready...")
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
		fmt.Printf("Got event: %s", event.Type)

		switch event.Type {
		default:
			deploy := event.Object.(*v1.Deployment)
			if deploy.Status.ReadyReplicas == deploy.Status.Replicas {
				fmt.Println("Deployment is ready")
				return nil
			}
		case watch.Deleted:
			return nil
		}
	}
}
