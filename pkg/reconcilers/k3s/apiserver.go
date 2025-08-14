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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	clog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ServerName                 = "k3s-server"
	ServerDockerImage          = "rancher/k3s"
	StorageDataName            = "k3s-data"
	StorageKubeconfigName      = "k3s-config"
	StorageClassName           = "standard" // kind default storage class name which is rancher/local-storage (same as k3s but different name)
	StorageMountPath           = "/var/lib/rancher/k3s"
	StorageKubeconfigMountPath = "/etc/rancher/k3s" // directory
	APIServerPort              = 6443               // k3s apiserver port
)

type Server struct {
	*shared.BaseReconciler
	ServerObject                 *appsv1.StatefulSet
	KubeconfigStorageClaimObject *v1.PersistentVolumeClaim
	DataStorageClaimObject       *v1.PersistentVolumeClaim
}

// build labels for k3s apiserver
func serverLabels() map[string]string {
	return map[string]string{
		"controller.kubeflex.dev/type":         string(tenancyv1alpha1.ControlPlaneTypeK3s),
		"controller.kubeflex.dev/service-name": ServiceName,
		"controller.kubeflex.dev/pvc-name":     StorageDataName,
	}
}

// build container image with tag of k3s apiserver
// see https://hub.docker.com/r/rancher/k3s/tags
func containerImage() string {
	imageTag := "v1.30.13-k3s1" // To update
	return fmt.Sprintf("%s:%s", ServerDockerImage, imageTag)
}

// serverTLSSAN returns the TLS SAN value expected by k3s server command
func serverTLSSAN(cpName string, cfg *shared.SharedConfig) string {
	return "--tls-san=" + GetClusterStaticDNSRecord(cpName, cfg)
}

func serverDataDir(path string) string {
	return "--data-dir=" + path
}

// NewServer return Server object
func NewServer(br *shared.BaseReconciler) *Server {
	return &Server{
		BaseReconciler:               br,
		ServerObject:                 &appsv1.StatefulSet{},
		KubeconfigStorageClaimObject: &v1.PersistentVolumeClaim{},
		DataStorageClaimObject:       &v1.PersistentVolumeClaim{},
	}
}

// Prepare k3s server object and its manifest
// NOTE: $cpName is used only for object Namespace, not its Name
func (r *Server) Prepare(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) error {
	log := clog.FromContext(ctx)
	cfg, err := r.GetConfig(ctx)
	if err != nil {
		log.Error(err, "failed to get shared config")
		return err
	}
	r.ServerObject = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(ServerName),                   //	always be unique as it has it dedicated namespace
			Namespace: ComputeSystemNamespaceName(hcp.Name), // must be dedicated name
			Labels:    serverLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: serverLabels(),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   string(ServerName),
					Labels: serverLabels(),
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            string(ServerName),
							Image:           containerImage(),
							ImagePullPolicy: v1.PullIfNotPresent,
							// Command: is by default `/bin/k3s`
							Args: []string{
								"server",
								serverTLSSAN(hcp.Name, cfg),
								serverDataDir(StorageMountPath),
							}, Env: []v1.EnvVar{
								{Name: "K3S_CONTROLPLANE_SECRET_NAME", Value: KubeconfigSecretName},
								{Name: "K3S_DATA_DIR", Value: StorageMountPath},
							},

							Ports: []v1.ContainerPort{
								{ContainerPort: shared.SecurePort},
							},
							VolumeMounts: []v1.VolumeMount{
								// VolumeMount k3s data
								{
									Name:      StorageDataName,
									MountPath: StorageMountPath,
									ReadOnly:  false,
								},
								// VolumeMount k3s kubeconfig
								{
									Name:      StorageKubeconfigName,
									MountPath: StorageKubeconfigMountPath,
									ReadOnly:  false,
								},
							},
							SecurityContext: &v1.SecurityContext{
								// Required by $ServerDockerImage
								Privileged: ptr.To(true),
								// Required to write filesystem as privileged
								ReadOnlyRootFilesystem: ptr.To(false),
								// Additional security
								SeccompProfile: &v1.SeccompProfile{
									Type: v1.SeccompProfileTypeRuntimeDefault,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						// Volume k3s data
						{
							Name: StorageDataName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: StorageDataName,
								},
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
					RestartPolicy: "Always",
				},
			},
		},
	}
	r.KubeconfigStorageClaimObject = computeNewPVC(StorageKubeconfigName, ComputeSystemNamespaceName(hcp.Name))
	r.DataStorageClaimObject = computeNewPVC(StorageDataName, ComputeSystemNamespaceName(hcp.Name))
	return nil
}

// computeNewPVC return PVC manifest with a given name and namespace
func computeNewPVC(name string, namespace string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    serverLabels(),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: ptr.To(StorageClassName),
		},
	}
}

// Reconcile k3s server
// implements ControlPlaneReconciler
func (r *Server) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	if err := r.Prepare(ctx, hcp); err != nil {
		return ctrl.Result{}, err
	}
	// Reconcile k3s pvc that is required for k3s server to run
	if result, err := r.reconcilePVC(ctx, hcp, r.KubeconfigStorageClaimObject); err != nil {
		return result, err
	}
	if result, err := r.reconcilePVC(ctx, hcp, r.DataStorageClaimObject); err != nil {
		return result, err
	}
	log.Info("reconcile k3s server")
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(r.ServerObject), r.ServerObject)
	switch {
	case err == nil:
		log.Info("k3s server is already created")
	case apierrors.IsNotFound(err):
		log.Error(err, "is not found error")
		// Set owner reference of the API server object
		if err := controllerutil.SetControllerReference(hcp, r.ServerObject, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference failed")
			return ctrl.Result{}, err
		}
		// Create the k3s server
		if err = r.Client.Create(ctx, r.ServerObject); err != nil {
			log.Error(err, "r.Client.Create failed")
			return ctrl.Result{RequeueAfter: 10}, err
		}
		log.Info("k3s server is succesfully created")
	default:
		log.Error(err, "failed to reconcile k3s server")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// reconcilePVC reconcile PVC tied to k3s server
func (r *Server) reconcilePVC(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane, pvc *v1.PersistentVolumeClaim) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	log.Info(" reconcile k3s PVC for server", "pvc", pvc.Name)
	err := r.Client.Get(ctx, client.ObjectKeyFromObject(pvc), pvc)
	switch {
	case err == nil:
		log.Info("pvc is already created", "pvc", pvc.Name)
	case apierrors.IsNotFound(err):
		log.Error(err, "pvc is not found error")
		if err := controllerutil.SetControllerReference(hcp, pvc, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference on pvc failed")
			return ctrl.Result{}, err
		}
		if err = r.Client.Create(ctx, pvc); err != nil {
			log.Error(err, "r.Client.Create pvc failed")
			return ctrl.Result{}, err
		}
		log.Info("pvc is succesfully created for k3s", "pvc", pvc.Name)
	default:
		log.Error(err, "failed to reconcile pvc", "pvc", pvc.Name)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
