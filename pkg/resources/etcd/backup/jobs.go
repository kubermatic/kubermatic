/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"strconv"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/resources"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	// BackupConfigNameLabelKey is the label key which should be used to name the BackupConfig a job belongs to.
	BackupConfigNameLabelKey = "backupConfig"
	// SharedVolumeName is the name of the `emptyDir` volume the initContainer
	// will write the backup to.
	SharedVolumeName = "etcd-backup"
	// clusterEnvVarKey defines the environment variable key for the cluster name.
	AccessKeyIDEnvVarKey = "ACCESS_KEY_ID"
	// SecretAccessKeyEnvVarKey defines the environment variable key for the backup credentials secret access key.
	SecretAccessKeyEnvVarKey = "SECRET_ACCESS_KEY"
	// bucketNameEnvVarKey defines the environment variable key for the backup bucket name.

	// BackupJobLabel defines the label we use on all backup jobs.
	BackupJobLabel   = "kubermatic-etcd-backup"
	clusterEnvVarKey = "CLUSTER"
	// BackupToCreateEnvVarKey defines the environment variable key for the name of the backup to create.
	BackupToCreateEnvVarKey = "BACKUP_TO_CREATE"
	// BackupToDeleteEnvVarKey defines the environment variable key for the name of the backup to delete.
	BackupToDeleteEnvVarKey = "BACKUP_TO_DELETE"
	// BackupScheduleEnvVarKey defines the environment variable key for the backup schedule.
	BackupScheduleEnvVarKey = "BACKUP_SCHEDULE"
	// BackupKeepCountEnvVarKey defines the environment variable key for the number of backups to keep.
	BackupKeepCountEnvVarKey = "BACKUP_KEEP_COUNT"
	// backupConfigEnvVarKey defines the environment variable key for the name of the backup configuration resource.
	BackupConfigEnvVarKey = "BACKUP_CONFIG"
	// AccessKeyIdEnvVarKey defines the environment variable key for the backup credentials access key id.
	BucketNameEnvVarKey = "BUCKET_NAME"
	// BackupEndpointEnvVarKey defines the environment variable key for the backup endpoint.
	BackupEndpointEnvVarKey = "ENDPOINT"
	// BackupInsecureEnvVarKey defines the environment variable key for a boolean that tells whether the
	// configured endpoint uses HTTPS ("false") or HTTP ("true").
	BackupInsecureEnvVarKey = "INSECURE"
)

type etcdBackupData interface {
	Cluster() *kubermaticv1.Cluster
	EtcdBackupDestination() *kubermaticv1.BackupDestination
	EtcdBackupStoreContainer() *corev1.Container
	EtcdBackupDeleteContainer() *corev1.Container
	EtcdLauncherImage() string
	EtcdLauncherTag() string
	GetClusterRef() metav1.OwnerReference
}

func BackupJob(data etcdBackupData, config *kubermaticv1.EtcdBackupConfig, status *kubermaticv1.BackupStatus) *batchv1.Job {
	storeContainer := data.EtcdBackupStoreContainer().DeepCopy()

	// If destination is set, we need to set the credentials and backup bucket details to match the destination
	if data.EtcdBackupDestination() != nil {
		storeContainer.Env = setEnvVar(storeContainer.Env, GenSecretEnvVar(AccessKeyIDEnvVarKey, AccessKeyIDEnvVarKey, data.EtcdBackupDestination()))
		storeContainer.Env = setEnvVar(storeContainer.Env, GenSecretEnvVar(SecretAccessKeyEnvVarKey, SecretAccessKeyEnvVarKey, data.EtcdBackupDestination()))
		storeContainer.Env = setEnvVar(storeContainer.Env, corev1.EnvVar{
			Name:  BucketNameEnvVarKey,
			Value: data.EtcdBackupDestination().BucketName,
		})
		storeContainer.Env = setEnvVar(storeContainer.Env, corev1.EnvVar{
			Name:  BackupEndpointEnvVarKey,
			Value: data.EtcdBackupDestination().Endpoint,
		})

		insecure := "false"
		if isInsecureURL(data.EtcdBackupDestination().Endpoint) {
			insecure = "true"
		}

		storeContainer.Env = setEnvVar(storeContainer.Env, corev1.EnvVar{
			Name:  BackupInsecureEnvVarKey,
			Value: insecure,
		})
	}

	storeContainer.Env = append(
		storeContainer.Env,
		corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: data.Cluster().Name,
		},
		corev1.EnvVar{
			Name:  BackupToCreateEnvVarKey,
			Value: status.BackupName,
		},
		corev1.EnvVar{
			Name:  BackupScheduleEnvVarKey,
			Value: config.Spec.Schedule,
		},
		corev1.EnvVar{
			Name:  BackupKeepCountEnvVarKey,
			Value: strconv.Itoa(config.GetKeptBackupsCount()),
		},
		corev1.EnvVar{
			Name:  BackupConfigEnvVarKey,
			Value: config.Name,
		})

	storeContainer.VolumeMounts = append(storeContainer.VolumeMounts, corev1.VolumeMount{
		Name:      "ca-bundle",
		MountPath: "/etc/ca-bundle/",
		ReadOnly:  true,
	})

	job := jobBase(config, data.Cluster(), status.JobName)

	job.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("%s-%s", rbac.EtcdLauncherServiceAccountName, data.Cluster().Name)
	job.Spec.Template.Spec.Containers = []corev1.Container{*storeContainer}
	job.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:    "backup-creator",
			Image:   fmt.Sprintf("%s:%s", data.EtcdLauncherImage(), data.EtcdLauncherTag()),
			Command: snapshotCommand(data.Cluster()),
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      SharedVolumeName,
					MountPath: "/backup",
				},
				{
					Name:      GetEtcdBackupSecretName(data.Cluster()),
					MountPath: "/etc/etcd/pki/client",
				},
				{
					Name:      "ca-bundle",
					MountPath: "/etc/ca-bundle/",
					ReadOnly:  true,
				},
			},
		},
	}

	job.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: SharedVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: GetEtcdBackupSecretName(data.Cluster()),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: GetEtcdBackupSecretName(data.Cluster()),
				},
			},
		},
		{
			Name: "ca-bundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.BackupCABundleConfigMapName(data.Cluster()),
					},
				},
			},
		},
	}

	return job
}

func snapshotCommand(cluster *kubermaticv1.Cluster) []string {
	return []string{
		"/etcd-launcher",
		"snapshot",
		"--etcd-ca-file=/etc/etcd/pki/client/ca.crt",
		"--etcd-client-cert-file=/etc/etcd/pki/client/backup-etcd-client.crt",
		"--etcd-client-key-file=/etc/etcd/pki/client/backup-etcd-client.key",
		fmt.Sprintf("--cluster=%s", cluster.Name),
		"--file=/backup/snapshot.db.gz",
		"--compress=gzip",
	}
}

func setEnvVar(envVars []corev1.EnvVar, newEnvVar corev1.EnvVar) []corev1.EnvVar {
	for i, envVar := range envVars {
		if strings.EqualFold(envVar.Name, newEnvVar.Name) {
			envVars[i] = newEnvVar
			return envVars
		}
	}
	envVars = append(envVars, newEnvVar)
	return envVars
}

func BackupDeleteJob(data etcdBackupData, config *kubermaticv1.EtcdBackupConfig, status *kubermaticv1.BackupStatus) *batchv1.Job {
	deleteContainer := data.EtcdBackupDeleteContainer().DeepCopy()

	// If destination is set, we need to set the credentials and backup bucket details to match the destination
	if data.EtcdBackupDestination() != nil {
		deleteContainer.Env = setEnvVar(deleteContainer.Env, GenSecretEnvVar(AccessKeyIDEnvVarKey, AccessKeyIDEnvVarKey, data.EtcdBackupDestination()))
		deleteContainer.Env = setEnvVar(deleteContainer.Env, GenSecretEnvVar(SecretAccessKeyEnvVarKey, SecretAccessKeyEnvVarKey, data.EtcdBackupDestination()))
		deleteContainer.Env = setEnvVar(deleteContainer.Env, corev1.EnvVar{
			Name:  BucketNameEnvVarKey,
			Value: data.EtcdBackupDestination().BucketName,
		})
		deleteContainer.Env = setEnvVar(deleteContainer.Env, corev1.EnvVar{
			Name:  BackupEndpointEnvVarKey,
			Value: data.EtcdBackupDestination().Endpoint,
		})

		insecure := "false"
		if isInsecureURL(data.EtcdBackupDestination().Endpoint) {
			insecure = "true"
		}

		deleteContainer.Env = setEnvVar(deleteContainer.Env, corev1.EnvVar{
			Name:  BackupInsecureEnvVarKey,
			Value: insecure,
		})
	}

	deleteContainer.Env = append(
		deleteContainer.Env,
		corev1.EnvVar{
			Name:  clusterEnvVarKey,
			Value: data.Cluster().Name,
		},
		corev1.EnvVar{
			Name:  BackupToDeleteEnvVarKey,
			Value: status.BackupName,
		},
		corev1.EnvVar{
			Name:  BackupScheduleEnvVarKey,
			Value: config.Spec.Schedule,
		},
		corev1.EnvVar{
			Name:  BackupKeepCountEnvVarKey,
			Value: strconv.Itoa(config.GetKeptBackupsCount()),
		},
		corev1.EnvVar{
			Name:  BackupConfigEnvVarKey,
			Value: config.Name,
		})

	deleteContainer.VolumeMounts = append(deleteContainer.VolumeMounts, corev1.VolumeMount{
		Name:      "ca-bundle",
		MountPath: "/etc/ca-bundle/",
		ReadOnly:  true,
	})

	job := jobBase(config, data.Cluster(), status.DeleteJobName)
	job.Spec.Template.Spec.Containers = []corev1.Container{*deleteContainer}
	job.Spec.ActiveDeadlineSeconds = resources.Int64(4 * 60)
	job.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "ca-bundle",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.BackupCABundleConfigMapName(data.Cluster()),
					},
				},
			},
		},
	}
	return job
}

func jobBase(backupConfig *kubermaticv1.EtcdBackupConfig, cluster *kubermaticv1.Cluster, jobName string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{
				resources.AppLabelKey:    BackupJobLabel,
				BackupConfigNameLabelKey: backupConfig.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				resources.GetClusterRef(cluster),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          ptr.To[int32](3),
			Completions:           ptr.To[int32](1),
			Parallelism:           ptr.To[int32](1),
			ActiveDeadlineSeconds: resources.Int64(2 * 60),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
}
