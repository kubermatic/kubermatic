//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
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
	meteringToolName          = "kubermatic-metering"
	meteringDataName          = "metering-data"
	meteringCronJobWeeklyName = "kubermatic-metering-report-weekly"
)

// deploymentCreator creates a new metering tool deployment per seed cluster.
func deploymentCreator(seed *kubermaticv1.Seed) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		//TODO: Add custom values for the metering deployment fields such as seed, interval and output-rotation.
		return meteringToolName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"role": meteringToolName,
				},
			}

			d.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType

			d.Spec.Template.Labels = map[string]string{
				"role": meteringToolName,
			}
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8080",
				"fluentbit.io/parser":  "glog",
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
										Name: SecretName,
									},
									Key: Endpoint,
								},
							},
						},
						{
							Name: "S3_BUCKET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: Bucket,
								},
							},
						},
						{
							Name: "ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: AccessKey,
								},
							},
						},
						{
							Name: "SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: SecretKey,
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
						"WEEKLY",
						"--seed",
						seed.Name,
					},
					Image:           "quay.io/kubermatic/metering:v0.5",
					ImagePullPolicy: corev1.PullAlways,
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 2112,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					LivenessProbe: &corev1.Probe{
						InitialDelaySeconds: 15,
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8081),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						PeriodSeconds:    20,
						SuccessThreshold: 1,
						TimeoutSeconds:   5,
						FailureThreshold: 3,
					},
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 5,
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(8081),
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
mc mb --ignore-existing s3/$S3_BUCKET
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
										Name: SecretName,
									},
									Key: Endpoint,
								},
							},
						},
						{
							Name: "S3_BUCKET",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: Bucket,
								},
							},
						},
						{
							Name: "ACCESS_KEY_ID",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: AccessKey,
								},
							},
						},
						{
							Name: "SECRET_ACCESS_KEY",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: SecretName,
									},
									Key: SecretKey,
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
			return d, nil
		}
	}
}
