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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

// CronJobCreator returns the func to create/update the etcd defragger cronjob
func CronJobCreator(seedName string) reconciling.NamedCronJobCreatorGetter {
	return func() (string, reconciling.CronJobCreator) {
		return meteringCronJobMonthly, func(job *batchv1beta1.CronJob) (*batchv1beta1.CronJob, error) {

			job.Spec.Schedule = "0 6 1 * *"
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
                                                          -last-month \
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
