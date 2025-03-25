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

package etcd

import (
	"fmt"
	"path/filepath"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type cronJobReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	RewriteImage(string) (string, error)
	GetClusterRef() metav1.OwnerReference
	EtcdLauncherImage() string
	EtcdLauncherTag() string
}

// CronJobReconciler returns the func to create/update the etcd defragger cronjob.
func CronJobReconciler(data cronJobReconcilerData) reconciling.NamedCronJobReconcilerFactory {
	return func() (string, reconciling.CronJobReconciler) {
		return resources.EtcdDefragCronJobName, func(job *batchv1.CronJob) (*batchv1.CronJob, error) {
			job.Name = resources.EtcdDefragCronJobName
			job.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
			job.Spec.SuccessfulJobsHistoryLimit = ptr.To[int32](1)
			job.Spec.Schedule = "@every 3h"

			job.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = rbac.EtcdLauncherServiceAccountName
			job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
			job.Spec.JobTemplate.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			job.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "defragger",
					Image:   fmt.Sprintf("%s:%s", data.EtcdLauncherImage(), data.EtcdLauncherTag()),
					Command: defraggerCommand(data),
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.ApiserverEtcdClientCertificateSecretName,
							MountPath: "/etc/etcd/pki/client",
							ReadOnly:  true,
						},
					},
				},
			}

			job.Spec.JobTemplate.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: resources.ApiserverEtcdClientCertificateSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.ApiserverEtcdClientCertificateSecretName,
						},
					},
				},
			}

			return job, nil
		}
	}
}

func defraggerCommand(data cronJobReconcilerData) []string {
	return []string{
		"/etcd-launcher",
		"defrag",
		"--etcd-ca-file=/etc/etcd/pki/client/ca.crt",
		fmt.Sprintf("--etcd-client-cert-file=%s", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateCertSecretKey)),
		fmt.Sprintf("--etcd-client-key-file=%s", filepath.Join("/etc/etcd/pki/client", resources.ApiserverEtcdClientCertificateKeySecretKey)),
		fmt.Sprintf("--cluster=%s", data.Cluster().Name),
	}
}
