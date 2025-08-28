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

package shared

import (
	"context"
	"fmt"

	"github.com/kubestellar/kubeflex/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tenancyv1alpha1 "github.com/kubestellar/kubeflex/api/v1alpha1"
)

const (
	jobName   = "update-cluster-info"
	baseImage = "ghcr.io/kubestellar/kubeflex/cmupdate"
)

func (r *BaseReconciler) ReconcileUpdateClusterInfoJob(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, cfg *SharedConfig, version string) error {
	namespace := util.GenerateNamespaceFromControlPlaneName(hcp.Name)

	// create job object
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
	}

	kubeconfigSecret := util.GetKubeconfSecretNameByControlPlaneType(string(hcp.Spec.Type))
	kubeconfigSecretKey := util.GetKubeconfSecretKeyNameByControlPlaneType(string(hcp.Spec.Type))

	err := r.Client.Get(ctx, client.ObjectKeyFromObject(job), job, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			job := generateClusterInfoJob(jobName, namespace, kubeconfigSecret, kubeconfigSecretKey, r.Version, cfg)
			if err := controllerutil.SetControllerReference(hcp, job, r.Scheme); err != nil {
				return fmt.Errorf("failed to SetControllerReference: %w", err)
			}
			if err = r.Client.Create(ctx, job, &client.CreateOptions{}); err != nil {
				if util.IsTransientError(err) {
					return err // Retry transient errors
				}
				return fmt.Errorf("failed to create job: %w", err)
			}
		} else if util.IsTransientError(err) {
			return err // Retry transient errors
		} else {
			return fmt.Errorf("failed to get job: %w", err)
		}
	}
	return nil
}

func generateClusterInfoJob(name, namespace, kubeconfigSecret, kubeconfigSecretKey, version string, cfg *SharedConfig) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           buildImageRef(version),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name: "KUBERNETES_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "KUBECONFIG_SECRET",
									Value: kubeconfigSecret,
								},
								{
									Name:  "KUBECONFIG_SECRET_KEY",
									Value: kubeconfigSecretKey,
								},
								{
									Name:  "HOST_CONTAINER",
									Value: cfg.HostContainer,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	if cfg.ExternalURL != "" {
		env := corev1.EnvVar{
			Name:  "EXTERNAL_URL",
			Value: cfg.ExternalURL,
		}
		job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, env)
	}
	return job
}

func buildImageRef(version string) string {
	tag := "latest"
	if version != "" {
		tag = util.ParseVersionNumber(version)
	}
	return fmt.Sprintf("%s:%s", baseImage, tag)
}
