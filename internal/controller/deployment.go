package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"mcc.ibm.org/kubeflex/pkg/util"
)

const (
	APIServerDeploymentName = "kube-apiserver"
	securePort              = 9443
	// temp values - to be injected by operator
	DBPassword    = "FAKE"
	DBReleaseName = "mypsql"
	DBNamespace   = "vks-system"
)

func (r *ControlPlaneReconciler) ReconcileAPIServerDeployment(ctx context.Context, name string, owner *metav1.OwnerReference) error {
	_ = clog.FromContext(ctx)
	namespace := util.GenerateNamespaceFromControlPlaneName(name)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIServerDeploymentName,
			Namespace: namespace,
		},
	}

	err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(deployment), deployment, &client.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			util.EnsureOwnerRef(deployment, owner)
			err = r.Client.Create(context.TODO(), generateDeployment(namespace, name), &client.CreateOptions{})
			if err != nil {
				return err
			}
		}
		return err
	}
	return nil
}

func generateDeployment(namespace, dbName string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIServerDeploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"component": "kube-apiserver",
				"tier":      "control-plane",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kube-apiserver",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kube-apiserver",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "kine",
							Image:   "rancher/kine:v0.9.9-amd64",
							Command: []string{"kine", "--endpoint", fmt.Sprintf("postgres://postgres:%s@%s-postgresql.%s.svc/%s?sslmode=disable", DBPassword, DBReleaseName, DBNamespace, dbName)},
							Ports: []v1.ContainerPort{{
								ContainerPort: 2379,
							}},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
							},
						},
						{
							Name:            "kube-apiserver",
							Image:           "registry.k8s.io/kube-apiserver:v1.27.1",
							ImagePullPolicy: v1.PullIfNotPresent,
							Command: []string{
								"kube-apiserver",
								"--allow-privileged=true",
								"--authorization-mode=Node,RBAC",
								"--client-ca-file=/etc/kubernetes/pki/ca.crt",
								"--enable-admission-plugins=NodeRestriction",
								"--enable-bootstrap-token-auth=true",
								"--etcd-servers=http://127.0.0.1:2379",
								"--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt",
								"--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key",
								"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
								"--proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt",
								"--proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key",
								"--requestheader-allowed-names=front-proxy-client",
								"--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt",
								"--requestheader-extra-headers-prefix=X-Remote-Extra-",
								"--requestheader-group-headers=X-Remote-Group",
								"--requestheader-username-headers=X-Remote-User",
								fmt.Sprintf("--secure-port=%d", securePort),
								"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
								"--service-account-key-file=/etc/kubernetes/pki/sa.pub",
								"--service-account-signing-key-file=/etc/kubernetes/pki/sa.key",
								"--service-cluster-ip-range=10.96.0.0/12",
								"--tls-cert-file=/etc/kubernetes/pki/apiserver.crt",
								"--tls-private-key-file=/etc/kubernetes/pki/apiserver.key",
							},
							Ports: []v1.ContainerPort{{
								ContainerPort: securePort,
							}},
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
								Requests: v1.ResourceList{
									"cpu":    resource.MustParse("500m"),
									"memory": resource.MustParse("512Mi"),
								},
							},
							LivenessProbe: &v1.Probe{
								FailureThreshold: 8,
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   "/livez",
										Port:   intstr.FromInt(securePort),
										Scheme: v1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								TimeoutSeconds:      15,
							},
							ReadinessProbe: &v1.Probe{
								FailureThreshold: 3,
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   "/readyz",
										Port:   intstr.FromInt(securePort),
										Scheme: v1.URISchemeHTTPS,
									},
								},
								PeriodSeconds:  1,
								TimeoutSeconds: 15,
							},
							StartupProbe: &v1.Probe{
								FailureThreshold: 24,
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path:   "/livez",
										Port:   intstr.FromInt(securePort),
										Scheme: v1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								TimeoutSeconds:      15,
							},
							VolumeMounts: []v1.VolumeMount{{
								MountPath: "/etc/kubernetes/pki",
								Name:      "k8s-certs",
								ReadOnly:  true,
							}},
						},
					},
					Volumes: []v1.Volume{{
						Name: "k8s-certs",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								SecretName: "k8s-certs",
							},
						},
					}},
				},
			},
		},
	}
	return deployment
}
