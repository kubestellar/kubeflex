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
	_ "embed"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

// Job bootstrapping k3s configmap server
type Job struct {
	*shared.BaseReconciler
}

const (
	JobName            = "k3s-bootstrap-kubeconfig"
	bashContainerImage = "bash:5"
)

func NewJob(namespace string) (*batchv1.Job, error) {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JobName,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Completions:  ptr.To(int32(1)),
			BackoffLimit: ptr.To(int32(5)),
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: namespace,
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyOnFailure,
					Containers: []v1.Container{
						{
							Name:  "executer",
							Image: bashContainerImage,
							Command: []string{
								"bash",
							},
							Args: []string{
								"./scripts/" + ScriptSaveKubeconfigIntoSecretName,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      ScriptsConfigMapName,
									MountPath: "/scripts",
									ReadOnly:  true,
								},
								{
									Name:      StorageKubeconfigName,
									MountPath: StorageKubeconfigMountPath,
									ReadOnly:  false,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: ScriptsConfigMapName,
							VolumeSource: v1.VolumeSource{
								ConfigMap: ptr.To(v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: ScriptsConfigMapName,
									},
								}),
							},
						},
						{
							Name: StorageKubeconfigName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: StorageKubeconfigName,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

// Reconcile the boostrap job
// implements Reconciler
func (r *Job) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	// Get k3s job tr is required for k3s server to run
	job, _ := NewJob(GenerateSystemNamespaceName(hcp.Name))
	log.Info("reconcile k3s job for server")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(job), job)
	if err != nil {
		log.Error(err, "get k3s job failed")
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s job is not found error")
			log.Info("k3s SetControllerReference on job")
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, job, r.Scheme); err != nil {
				log.Error(err, "k3s SetControllerReference job failed")
				return ctrl.Result{}, err
			}
			// Create k3s job on cluster
			log.Info("create k3s job on cluster", "job", job)
			if err = r.Client.Create(ctx, job); err != nil {
				log.Error(err, "k3s creation of job failed")
				return ctrl.Result{RequeueAfter: 10}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
