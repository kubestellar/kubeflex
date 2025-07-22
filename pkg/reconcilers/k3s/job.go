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
	_ "embed"

	"github.com/kubestellar/kubeflex/pkg/reconcilers/shared"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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
