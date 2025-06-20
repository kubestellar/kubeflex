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

type BootstrapSecretJob struct {
	*shared.BaseReconciler
	Object *batchv1.Job
}

const (
	JobName            = "k3s-bootstrap-kubeconfig"
	bashContainerImage = "bash:5"
)

// NewBootstrapSecretJob create job to booststrap k3s kubeconfig into secret
func NewBootstrapSecretJob(br *shared.BaseReconciler) *BootstrapSecretJob {
	return &BootstrapSecretJob{
		BaseReconciler: br,
		Object:         &batchv1.Job{},
	}
}

// Prepare job object and its manifest
func (r *BootstrapSecretJob) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	log := clog.FromContext(ctx)
	cfg, err := r.GetConfig(ctx)
	if err != nil {
		log.Error(err, "failed to load shared config")
		return err
	}
	namespace := ComputeSystemNamespaceName(hcp.Name)
	ingressDNS := GetClusterServerURI(hcp.Name, cfg)
	serviceDNS := GetInClusterStaticDNSRecord(namespace)
	r.Object = &batchv1.Job{
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
							Name:  "executer-1",
							Image: bashContainerImage,
							Command: []string{
								"bash",
							},
							Args: []string{
								"./scripts/" + ScriptSaveKubeconfigIntoSecretName,
							},
							Env: []v1.EnvVar{
								{Name: "DNS_SVC", Value: serviceDNS},
								{Name: "DNS_INGRESS", Value: ingressDNS},
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
						{
							Name:  "executer-2",
							Image: bashContainerImage,
							Command: []string{
								"bash",
							},
							Args: []string{
								"./scripts/" + ScriptSaveTokenIntoSecretName,
							},
							Env: []v1.EnvVar{
								{Name: "K3S_CONTROLPLANE_SECRET_NAME", Value: KubeconfigSecretName},
								{Name: "K3S_DATA_DIR", Value: StorageMountPath},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      ScriptsConfigMapName,
									MountPath: "/scripts",
									ReadOnly:  true,
								},
								{
									Name:      StorageDataName,
									MountPath: StorageMountPath,
									ReadOnly:  true,
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
						{
							Name: StorageDataName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: StorageDataName,
								},
							},
						},
					},
				},
			},
		},
	}
	return nil
}

// Reconcile the boostrap job
// implements Reconciler
func (r *BootstrapSecretJob) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("reconcile k3s job for server")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.Object), r.Object)
	switch {
	case err == nil:
		log.Info("bootstrap secret job is already created", "job", r.Object.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "k3s job is not found error")
		log.Info("k3s SetControllerReference on job")
		if err := controllerutil.SetControllerReference(hcp, r.Object, r.Scheme); err != nil {
			log.Error(err, "k3s SetControllerReference job failed")
			return ctrl.Result{}, err
		}
		if err = r.Client.Create(ctx, r.Object); err != nil {
			log.Error(err, "k3s creation of job failed")
			return ctrl.Result{RequeueAfter: 10}, err
		}
	default:
		log.Error(err, "failed to reconcile bootstrap secret job")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
