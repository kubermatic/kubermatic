// +build ee

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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// cronJobCreator returns the func to create/update the etcd defragger cronjob
func cronJobCreator(seedName string) reconciling.NamedCronJobCreatorGetter {
	return func() (string, reconciling.CronJobCreator) {
		return meteringCronJobMonthly, func(job *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {

			job.Spec.Schedule = "0 6 * * 1"
			job.Spec.JobTemplate.Spec.Parallelism = pointer.Int32Ptr(1)
			job.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = meteringToolName
			job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			job.Spec.JobTemplate.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:  "s3fetch",
					Image: "docker.io/minio/mc:RELEASE.2021-07-27T06-46-19Z",
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						`mc config host add s3 $S3_ENDPOINT $ACCESS_KEY_ID $SECRET_ACCESS_KEY
mc mirror --newer-than "65d0h0m" s3/$S3_BUCKET /metering-data || true`,
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

			job.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "kubermatic-metering-report",
					Image: "quay.io/kubermatic/metering:7a12117",
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						`/usr/local/bin/kubermatic-metering-report -workdir=/metering-data \
                                                          -reportdir=/report \
                                                          -last-week \
                                                          -seed=` + seedName + ` \
                                                          -scrape-interval=300
                        touch /report/finished`,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workdir",
							MountPath: "/" + meteringDataName,
							ReadOnly:  true,
						},
						{
							Name:      "report-output",
							MountPath: "/report",
							ReadOnly:  false,
						},
					},
				},
				{
					Name:  "s3upload",
					Image: "docker.io/minio/mc:RELEASE.2021-07-27T06-46-19Z",
					Command: []string{
						"/bin/sh",
					},
					Args: []string{
						"-c",
						`while [ ! -f /report/finished ]
do
  sleep 10
done
rm /report/finished
mc config host add s3 $S3_ENDPOINT $ACCESS_KEY_ID $SECRET_ACCESS_KEY
mc mirror /report s3/$S3_BUCKET`,
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
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "report-output",
							MountPath: "/report",
							ReadOnly:  false,
						},
					},
				},
			}
			job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: resources.ImagePullSecretName,
				},
			}

			job.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "workdir",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "report-output",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			}

			return job, nil
		}
	}
}
