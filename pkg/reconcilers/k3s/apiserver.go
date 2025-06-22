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
	"github.com/kubestellar/kubeflex/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const APIServerPodName = "k3s-server"
const APIServerDockerImage = "rancher/k3s"

// K3s API server
// NOTE: k3s is a single binary containing apiserver, etcd, controller-manager... therefore `APIServer` refers to all components
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
	log := logf.FromContext(ctx)
	log.Info("k3s:apiserver:Reconcile: begin")
	// Get k3s server from hosting cluster and stored it in apiServerObject
	apiServerObject := &appsv1.StatefulSet{}
	apiServerObjectKey := client.ObjectKey{Namespace: util.GenerateNamespaceFromControlPlaneName(string(tenancyv1alpha1.ControlPlaneTypeK3s)), Name: APIServerPodName}
	err := r.Client.Get(ctx, apiServerObjectKey, apiServerObject)
	if err != nil {
		r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, err) // TODO: to change
		// is NotFound, we retry in 5s
		if apierrors.IsNotFound(err) {
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, apiServerObject, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			// Create the k3s server
			// TODO create
			if err = r.Client.Create(ctx, apiServerObject); err != nil {
				// if not able to create, we retry in 10s
				// TODO implement exp. backoff?
				return ctrl.Result{RequeueAfter: 10}, err
			}
		}
		return ctrl.Result{}, err
	}
	return r.BaseReconciler.Reconcile(ctx, hcp) // TODO: to change
}
