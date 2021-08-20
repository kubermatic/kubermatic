/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package metering

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	meteringToolName       = "kubermatic-metering"
	meteringDataName       = "metering-data"
	meteringCronJobMonthly = "kubermatic-metering-report-monthly"
)

// MeteringToolDeploymentCreator creates a new metering tool deployment per seed cluster.
func MeteringToolDeploymentCreator(_ *kubermaticv1.Seed) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		//TODO: Add custom values for the metering deployment fields such as seed, interval and output-rotation.
		return meteringToolName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"role":    meteringToolName,
					"version": "1",
				},
			}

			d.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType

			d.Spec.Template.Labels = map[string]string{
				"role":    meteringToolName,
				"version": "1",
			}
			d.Spec.Template.Annotations = map[string]string{
				"kubermatic/scrape":      "true",
				"kubermatic/scrape_port": "2112",
				"fluentbit.io/parser":    "glog",
			}

			d.Spec.Template.Spec.ServiceAccountName = "kubermatic-metering"
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:    "s3fetch",
					Image:   "docker.io/minio/mc:RELEASE.2021-07-27T06-46-19Z",
					Command: []string{"/bin/sh"},
					Args: []string{
						"-c",
						`mc config host add s3 $S3_ENDPOINT $ACCESS_KEY_ID $SECRET_ACCESS_KEY
mc mirror --newer-than "32d0h0m" s3/$S3_BUCKET /metering-data || true`,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workdir",
							MountPath: "/" + meteringDataName,
							ReadOnly:  false,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name: "S3_ENDPOINT",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "endpoint",
								},
							},
						},
						{
							Name: "S3_BUCKET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "bucket",
								},
							},
						},
						{
							Name: "ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "accessKey",
								},
							},
						},
						{
							Name: "SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "secretKey",
								},
							},
						},
					},
				},
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "metering",
					Command: []string{"/usr/local/bin/kubermatic-metering"},
					Args: []string{
						"-workdir",
						"/" + meteringDataName,
						"-output-rotation",
						"MONTHLY",
					},
					Image:           "quay.io/kubermatic/metering:d2f3002",
					ImagePullPolicy: corev1.PullAlways,
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 2112,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/metrics",
								Port:   intstr.FromInt(2112),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   5,
						FailureThreshold: 3,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/metrics",
								Port:   intstr.FromInt(2112),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   5,
						FailureThreshold: 3,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workdir",
							MountPath: "/" + meteringDataName,
							ReadOnly:  false,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("3Gi"),
						},
					},
				},
				{
					Name: "s3upload",
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						`mc config host add s3 $S3_ENDPOINT $ACCESS_KEY_ID $SECRET_ACCESS_KEY
mc mb --ignore-existing --region=$S3_REGION s3/$S3_BUCKET
while true; do mc mirror --overwrite /metering-data s3/$S3_BUCKET; sleep 300; done`,
					},
					Image:           "docker.io/minio/mc:RELEASE.2021-07-27T06-46-19Z",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{
						{
							Name: "S3_ENDPOINT",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "endpoint",
								},
							},
						},
						{
							Name: "S3_REGION",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "region",
								},
							},
						},
						{
							Name: "S3_BUCKET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "bucket",
								},
							},
						},
						{
							Name: "ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "accessKey",
								},
							},
						},
						{
							Name: "SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "s3",
									},
									Key: "secretKey",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workdir",
							MountPath: "/" + meteringDataName,
							ReadOnly:  true,
						},
					},
				},
			}

			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: resources.ImagePullSecretName,
				},
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "workdir",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: meteringDataName,
						},
					},
				},
			}

			d.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "kubermatic.io/type",
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											"stable",
										},
									},
								},
							},
							Weight: 100,
						},
					},
				},
			}

			d.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Key:      "only_critical",
					Operator: corev1.TolerationOpEqual,
					Value:    "true",
				},
			}

			return d, nil
		}
	}
}
