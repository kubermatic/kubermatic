/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package rancherserver

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("2"),
		},
	}

	rancherServerDiskRequirement = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse("2Gi"),
		},
	}
)

// StatefulSetCreator returns the function to reconcile the StatefulSet
func StatefulSetCreator(data *resources.TemplateData) reconciling.NamedStatefulSetCreatorGetter {
	return func() (string, reconciling.StatefulSetCreator) {
		return resources.RancherStatefulSetName, func(set *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
			set.Name = resources.RancherStatefulSetName
			set.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			baseLabels := getBasePodLabels(data.Cluster())
			set.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			set.Spec.ServiceName = resources.RancherStatefulSetName
			set.Labels = baseLabels
			set.Spec.Replicas = resources.Int32(1)

			set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: baseLabels,
			}
			set.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name: resources.RancherStatefulSetName,
					// TODO this shouldn't be hardcoded
					Image:           "docker.io/rancher/rancher:v2.3.2",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args: []string{
						"--http-listen-port=80",
						"--https-listen-port=443",
						"--add-local=false",
						"--k8s-mode=embedded",
					},
					Env: []corev1.EnvVar{
						{
							Name: "CATTLE_NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "rancher-data",
							MountPath: "/var/lib/rancher/",
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(443),
								Scheme: "HTTPS",
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    30,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(443),
								Scheme: "HTTPS",
							},
						},
						InitialDelaySeconds: 30,
						FailureThreshold:    8,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: defaultResourceRequirements,
				},
			}

			if len(set.Spec.VolumeClaimTemplates) == 0 {
				set.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "rancher-data",
							OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							StorageClassName: resources.String("kubermatic-fast"),
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources:        rancherServerDiskRequirement,
						},
					},
				}
			}

			return set, nil
		}
	}
}

func getBasePodLabels(cluster *kubermaticv1.Cluster) map[string]string {
	additionalLabels := map[string]string{
		"cluster": cluster.Name,
	}
	return resources.BaseAppLabels(resources.RancherStatefulSetName, additionalLabels)
}
