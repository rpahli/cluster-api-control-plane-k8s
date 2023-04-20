/*
Copyright 2021 The Kubernetes Authors.

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

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "sigs.k8s.io/cluster-api-provider-nested/virtualcluster/pkg/apis/tenancy/v1alpha1"
)

func createClusterVersion(fns ...func(*v1alpha1.ClusterVersion)) *v1alpha1.ClusterVersion {
	instance := &v1alpha1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "clusterversion-sample",
		},
		Spec: *defaultClusterVersion,
	}

	for _, fn := range fns {
		fn(instance)
	}

	return instance
}

var _ = Describe("ClusterVersion Controller", func() {

	Context("Reconcile ClusterVersion Cluster", func() {
		It("Should create resources successfully", func() {
			ctx := context.TODO()
			Expect(cli).ShouldNot(BeNil())

			instance := createClusterVersion()
			Expect(cli.Create(ctx, instance)).Should(Succeed())

			objectKey := client.ObjectKeyFromObject(instance)

			By("Adding Finalizer")
			Eventually(func() bool {
				i := &v1alpha1.ClusterVersion{}
				err := cli.Get(ctx, objectKey, i)
				return err == nil && len(i.GetFinalizers()) == 1
			}, timeout, interval).Should(BeTrue())

			By("Deleting ClusterVersion")
			Expect(cli.Delete(ctx, instance)).To(BeNil())
		})
	})
})

var defaultClusterVersion = &v1alpha1.ClusterVersionSpec{
	ETCD: &v1alpha1.StatefulSetSvcBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: "etcd",
		},
		StatefulSet: &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "etcd",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:             pointer.Int32Ptr(1),
				RevisionHistoryLimit: pointer.Int32Ptr(10),
				ServiceName:          "etcd",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component-name": "etcd",
					},
				},
				UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.RollingUpdateStatefulSetStrategyType,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"component-name": "etcd",
						},
					},
					Spec: corev1.PodSpec{
						Subdomain: "etcd",
						Containers: []corev1.Container{
							{
								Name:            "etcd",
								Image:           "virtualcluster/etcd-v3.4.0",
								ImagePullPolicy: corev1.PullAlways,
								Command:         []string{"etcd"},
								Env: []corev1.EnvVar{
									{
										Name: "HOSTNAME",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "metadata.name",
											},
										},
									},
								},
								Args: []string{
									"--name=$(HOSTNAME)",
									"--trusted-ca-file=/etc/kubernetes/pki/root/tls.crt",
									"--client-cert-auth",
									"--cert-file=/etc/kubernetes/pki/etcd/tls.crt",
									"--key-file=/etc/kubernetes/pki/etcd/tls.key",
									"--peer-client-cert-auth",
									"--peer-trusted-ca-file=/etc/kubernetes/pki/root/tls.crt",
									"--peer-cert-file=/etc/kubernetes/pki/etcd/tls.crt",
									"--peer-key-file=/etc/kubernetes/pki/etcd/tls.key",
									"--listen-peer-urls=https://0.0.0.0:2380",
									"--listen-client-urls=https://0.0.0.0:2379",
									"--initial-advertise-peer-urls=https://$(HOSTNAME).etcd:2380",
									"--advertise-client-urls=https://$(HOSTNAME).etcd:2379",
									"--initial-cluster-state=new",
									"--initial-cluster-token=vc-etcd",
									"--data-dir=/var/lib/etcd/data",
								},
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"sh",
												"-c",
												"ETCDCTL_API=3 etcdctl --endpoints=https://etcd:2379 --cacert=/etc/kubernetes/pki/root/tls.crt --cert=/etc/kubernetes/pki/etcd/tls.crt --key=/etc/kubernetes/pki/etcd/tls.key endpoint health",
											},
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 60,
									TimeoutSeconds:      15,
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										Exec: &corev1.ExecAction{
											Command: []string{
												"sh",
												"-c",
												"ETCDCTL_API=3 etcdctl --endpoints=https://etcd:2379 --cacert=/etc/kubernetes/pki/root/tls.crt --cert=/etc/kubernetes/pki/etcd/tls.crt --key=/etc/kubernetes/pki/etcd/tls.key endpoint health",
											},
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 15,
									PeriodSeconds:       2,
									TimeoutSeconds:      15,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/etc/kubernetes/pki/etcd",
										Name:      "etcd-ca",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/pki/root",
										Name:      "root-ca",
										ReadOnly:  true,
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "etcd-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "etcd-ca",
									},
								},
							},
							{
								Name: "root-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "root-ca",
									},
								},
							},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{},
		},
		Service: &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "etcd",
			},
			Spec: corev1.ServiceSpec{
				PublishNotReadyAddresses: true,
				Type:                     corev1.ServiceTypeClusterIP,
				ClusterIP:                corev1.ClusterIPNone,
				Selector: map[string]string{
					"component-name": "etcd",
				},
			},
		},
	},
	APIServer: &v1alpha1.StatefulSetSvcBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apiserver",
		},
		StatefulSet: &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "apiserver",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas:             pointer.Int32Ptr(1),
				RevisionHistoryLimit: pointer.Int32Ptr(10),
				ServiceName:          "apiserver-svc",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component-name": "apiserver",
					},
				},
				UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.RollingUpdateStatefulSetStrategyType,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"component-name": "apiserver",
						},
					},
					Spec: corev1.PodSpec{
						Hostname:  "apiserver",
						Subdomain: "apiserver-svc",
						Containers: []corev1.Container{
							{
								Name:            "apiserver",
								Image:           "virtualcluster/apiserver-v1.16.2",
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"kube-apiserver",
								},
								Args: []string{
									"--bind-address=0.0.0.0",
									"--allow-privileged=true",
									"--anonymous-auth=true",
									"--client-ca-file=/etc/kubernetes/pki/root/tls.crt",
									"--tls-cert-file=/etc/kubernetes/pki/apiserver/tls.crt",
									"--tls-private-key-file=/etc/kubernetes/pki/apiserver/tls.key",
									"--kubelet-https=true",
									"--kubelet-client-certificate=/etc/kubernetes/pki/apiserver/tls.crt",
									"--kubelet-client-key=/etc/kubernetes/pki/apiserver/tls.key",
									"--enable-bootstrap-token-auth=true",
									"--etcd-servers=https://etcd-0.etcd:2379",
									"--etcd-cafile=/etc/kubernetes/pki/root/tls.crt",
									"--etcd-certfile=/etc/kubernetes/pki/apiserver/tls.crt",
									"--etcd-keyfile=/etc/kubernetes/pki/apiserver/tls.key",
									"--service-account-key-file=/etc/kubernetes/pki/service-account/tls.key",
									"--service-cluster-ip-range=10.32.0.0/16",
									"--service-node-port-range=30000-32767",
									"--authorization-mode=Node,RBAC",
									"--runtime-config=api/all",
									"--enable-admission-plugins=NamespaceLifecycle,NodeRestriction,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota",
									"--apiserver-count=1",
									"--enable-aggregator-routing=true",
									"--requestheader-client-ca-file=/etc/kubernetes/pki/root/tls.crt",
									"--requestheader-allowed-names=front-proxy-client",
									"--requestheader-username-headers=X-Remote-User",
									"--requestheader-group-headers=X-Remote-Group",
									"--requestheader-extra-headers-prefix=X-Remote-Extra-",
									"--proxy-client-key-file=/etc/kubernetes/pki/frontproxy/tls.key",
									"--proxy-client-cert-file=/etc/kubernetes/pki/frontproxy/tls.crt",
									"--v=2",
								},
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 6443,
										Name:          "api",
										Protocol:      corev1.ProtocolTCP, // Must be explicit in 1.19
									},
								},
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										TCPSocket: &corev1.TCPSocketAction{
											Port: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 6443,
											},
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 15,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Port: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 6443,
											},
											Path:   "/healthz",
											Scheme: "HTTPS",
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 5,
									PeriodSeconds:       2,
									TimeoutSeconds:      30,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/etc/kubernetes/pki/apiserver",
										Name:      "apiserver-ca",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/pki/frontproxy",
										Name:      "front-proxy-ca",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/pki/root",
										Name:      "root-ca",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/pki/service-account",
										Name:      "serviceaccount-rsa",
										ReadOnly:  true,
									},
								},
							},
						},
						DNSConfig: &corev1.PodDNSConfig{
							Searches: []string{
								"cluster.local",
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "apiserver-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "apiserver-ca",
									},
								},
							},
							{
								Name: "root-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "root-ca",
									},
								},
							},
							{
								Name: "front-proxy-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "front-proxy-ca",
									},
								},
							},
							{
								Name: "serviceaccount-rsa",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "serviceaccount-rsa",
									},
								},
							},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{},
		},
		Service: &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "apiserver-svc",
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"component-name": "apiserver",
				},
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Port:     6443,
						Protocol: corev1.ProtocolTCP,
						TargetPort: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: "api",
						},
					},
				},
			},
		},
	},
	ControllerManager: &v1alpha1.StatefulSetSvcBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: "controller-manager",
		},
		StatefulSet: &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "controller-manager",
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: "controller-manager-svc",
				Replicas:    pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"component-name": "controller-manager",
					},
				},
				UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.RollingUpdateStatefulSetStrategyType,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"component-name": "controller-manager",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "controller-manager",
								Image:           "virtualcluster/controller-manager-v1.16.2",
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"kube-controller-manager",
								},
								Args: []string{
									"--bind-address=0.0.0.0",
									"--cluster-cidr=10.200.0.0/16",
									"--cluster-signing-cert-file=/etc/kubernetes/pki/root/tls.crt",
									"--cluster-signing-key-file=/etc/kubernetes/pki/root/tls.key",
									"--kubeconfig=/etc/kubernetes/kubeconfig/controller-manager-kubeconfig",
									"--authorization-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager-kubeconfig",
									"--authentication-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager-kubeconfig",
									"--leader-elect=false",
									"--root-ca-file=/etc/kubernetes/pki/root/tls.crt",
									"--service-account-private-key-file=/etc/kubernetes/pki/service-account/tls.key",
									"--service-cluster-ip-range=10.32.0.0/24",
									"--use-service-account-credentials=true",
									"--experimental-cluster-signing-duration=87600h",
									"--node-monitor-grace-period=200s",
									"--controllers=*,-nodelifecycle",
									"--v=2",
								},
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 10252,
											},
											Scheme: "HTTP",
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 15,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Port: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 10252,
											},
											Path:   "/healthz",
											Scheme: "HTTP",
										},
									},
									FailureThreshold:    8,
									InitialDelaySeconds: 15,
									PeriodSeconds:       2,
									TimeoutSeconds:      15,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/etc/kubernetes/pki/root",
										Name:      "root-ca",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/pki/service-account",
										Name:      "serviceaccount-rsa",
										ReadOnly:  true,
									},
									{
										MountPath: "/etc/kubernetes/kubeconfig",
										Name:      "kubeconfig",
										ReadOnly:  true,
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "root-ca",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "root-ca",
									},
								},
							},
							{
								Name: "serviceaccount-rsa",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "serviceaccount-rsa",
									},
								},
							},
							{
								Name: "kubeconfig",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										DefaultMode: pointer.Int32Ptr(420),
										SecretName:  "controller-manager-kubeconfig",
									},
								},
							},
						},
					},
				},
			},
			Status: appsv1.StatefulSetStatus{},
		},
	},
}
