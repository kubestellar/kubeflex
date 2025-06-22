/*
Copyright 2025 The KubeStellar Authors.

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

package k3s

import (
	"context"
	"fmt"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const APIServerPodName = "k3s-apiserver"
const APIServerDockerImage = "rancher/k3s"

// K3s API server
type APIServer struct {
	*shared.BaseReconciler
}

// build labels for k3s apiserver
func apiServerLabels() map[string]string {
	return map[string]string{
		"controller.kubeflex.dev/type":         string(tenancyv1alpha1.ControlPlaneTypeK3s),
		"controller.kubeflex.dev/service-name": ServiceName,
	}
}

// build container image with tag of k3s apiserver
// see https://hub.docker.com/r/rancher/k3s/tags
func containerImage() string {
	imageTag := "v1.30.13-k3s1" // To update
	return fmt.Sprintf("%s:%s", APIServerDockerImage, imageTag)
}

// Init API server object to apply on kubernetes server
// TODO: to implement
func NewAPIServer() (_ metav1.Object, err error) {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   string(APIServerPodName),
			Labels: apiServerLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: apiServerLabels(),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   string(APIServerPodName),
					Labels: apiServerLabels(),
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            string(APIServerPodName),
							Image:           containerImage(),
							ImagePullPolicy: v1.PullIfNotPresent,
							// Command: is by default `/bin/k3s`
							Args: []string{
								"server",
							},
							Ports: []v1.ContainerPort{
								{ContainerPort: shared.SecurePort},
							},
						},
					},
					Volumes: []v1.Volume{
						// Volume kubernetes certificates
						{
							Name: "k8s-certs",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "k8s-certs",
								},
							},
						},
						// Volume ConfigMap kubeconfig
						{
							Name: "cm-kubeconfig",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "cm-kubeconfig",
								},
							},
						},
					},
					RestartPolicy: "Always",
				},
			},
		},
	}, nil
}

// Reconcile API server
// implements ControlPlaneReconciler
// TODO: to implement
func (r *APIServer) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	return r.BaseReconciler.Reconcile(ctx, hcp)
}
