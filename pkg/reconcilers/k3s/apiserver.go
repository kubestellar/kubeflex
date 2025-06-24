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

// k3s constants
const ServerName = "k3s-server"
const ServerDockerImage = "rancher/k3s"
const StorageName = "data-k3s-server"
const StorageClassName = "standard" // kind default storage class name which is rancher/local-storage (same as k3s but different name)
const StoragePath = "/data"

// K3s API server
// NOTE: k3s is a single binary containing apiserver, etcd, controller-manager... therefore `Server` refers to all components
type Server struct {
	*shared.BaseReconciler
}

// build labels for k3s apiserver
func serverLabels() map[string]string {
	return map[string]string{
		"controller.kubeflex.dev/type":         string(tenancyv1alpha1.ControlPlaneTypeK3s),
		"controller.kubeflex.dev/service-name": ServiceName,
		"controller.kubeflex.dev/pvc-name":     StorageName,
	}
}

// build container image with tag of k3s apiserver
// see https://hub.docker.com/r/rancher/k3s/tags
func containerImage() string {
	imageTag := "v1.30.13-k3s1" // To update
	return fmt.Sprintf("%s:%s", ServerDockerImage, imageTag)
}

// Init API server object to apply on controlplane $cpName
// NOTE: $cpName is used only for object Namespace, not its Name
func NewServer(cpName string) (*appsv1.StatefulSet, error) {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(ServerName),                  //	always be unique as it has it dedicated namespace
			Namespace: GenerateSystemNamespaceName(cpName), // must be dedicated name
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
							},
							Ports: []v1.ContainerPort{
								{ContainerPort: shared.SecurePort},
							},
							VolumeMounts: []v1.VolumeMount{
								// VolumeMount k3s data
								{
									Name:      StorageName,
									MountPath: StoragePath,
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
							Name: StorageName,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: StorageName,
								},
							},
						},
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

// apiVersion: v1
// kind: PersistentVolumeClaim
// metadata:
//
//	name: local-path-pvc
//	namespace: k3stest
//
// spec:
//
//	accessModes:
//	  - ReadWriteOnce
//	storageClassName: standard
//	volumeMode: Filesystem
//	resources:
//	  requests:
//	    storage: 2Gi
func NewPVC(cpName string) (*v1.PersistentVolumeClaim, error) {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StorageName,
			Namespace: GenerateSystemNamespaceName(cpName),
			Labels:    serverLabels(),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse("2Gi"),
				},
			},
			StorageClassName: ptr.To(StorageClassName),
		},
	}, nil
}

// Reconcile k3s server
// implements ControlPlaneReconciler
// TODO to implement
func (r *Server) Reconcile(ctx context.Context, hcp *tenancyv1alpha1.ControlPlane) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	// Get k3s pvc that is required for k3s server to run
	k3sPVCObject := &v1.PersistentVolumeClaim{}
	k3sPVCObjectKey := client.ObjectKey{Namespace: GenerateSystemNamespaceName(hcp.Name), Name: StorageName}
	log.Info("k3s:server.go:Reconcile:", "k3sPVCObjectKey", k3sPVCObjectKey)
	err := r.Client.Get(ctx, k3sPVCObjectKey, k3sPVCObject)
	if err != nil {
		log.Error(err, "k3s:server.go:Reconcile:r.Client.Get pvc failed")
		r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, err) // TODO: to change
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s:server.go:Reconcile:pvc is not found error")
			log.Info("k3s:server.go:Reconcile:call NewServer() on k3sServerObject")
			// Generate new k3s server
			k3sPVCObject, _ = NewPVC(hcp.Name)
			log.Info("k3s:server.go:Reconcile:call SetControllerReference on pvc")
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, k3sPVCObject, r.Scheme); err != nil {
				log.Error(err, "k3s:server.go:Reconcile:SetControllerReference on pvc failed")
				return ctrl.Result{}, err
			}
			// Create the k3s server
			log.Info("k3s:server.go:Reconcile:call r.Client.Create on", "k3sPVCObject", k3sPVCObject)
			if err = r.Client.Create(ctx, k3sPVCObject); err != nil {
				log.Error(err, "k3s:server.go:Reconcile:r.Client.Create pvc failed")
				return ctrl.Result{RequeueAfter: 10}, err
			}
		}
		return ctrl.Result{}, err
	}
	// Get k3s server from hosting cluster and stored it in k3sServerObject
	k3sServerObject := &appsv1.StatefulSet{}
	k3sServerObjectKey := client.ObjectKey{Namespace: GenerateSystemNamespaceName(hcp.Name), Name: ServerName}
	log.Info("k3s:server.go:Reconcile:", "k3sServerObjectKey", k3sServerObjectKey)
	err = r.Client.Get(ctx, k3sServerObjectKey, k3sServerObject)
	if err != nil {
		log.Error(err, "k3s:server.go:Reconcile:r.Client.Get failed")
		r.BaseReconciler.UpdateStatusForSyncingError(ctx, hcp, err) // TODO: to change
		if apierrors.IsNotFound(err) {
			log.Error(err, "k3s:server.go:Reconcile:is not found error")
			log.Info("k3s:server.go:Reconcile:call NewServer() on k3sServerObject")
			// Generate new k3s server
			k3sServerObject, _ = NewServer(hcp.Name)
			log.Info("k3s:server.go:Reconcile:call SetControllerReference")
			// Set owner reference of the API server object
			if err := controllerutil.SetControllerReference(hcp, k3sServerObject, r.Scheme); err != nil {
				log.Error(err, "k3s:server.go:Reconcile:SetControllerReference failed")
				return ctrl.Result{}, err
			}
			// Create the k3s server
			log.Info("k3s:server.go:Reconcile:call r.Client.Create on", "k3sServerObject", k3sServerObject)
			if err = r.Client.Create(ctx, k3sServerObject); err != nil {
				log.Error(err, "k3s:server.go:Reconcile:r.Client.Create failed")
				return ctrl.Result{RequeueAfter: 10}, err
			}
		}
		log.Info("k3s:server.go:Reconcile:end of reconcile k3s server")
		return ctrl.Result{}, err
	}
	// Update to success
	log.Info("k3s:server.go:Reconcile:reconcile is a success...")
	return r.BaseReconciler.Reconcile(ctx, hcp) // TODO: to change
}
