/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package encryption

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	EncryptionJobPrefix    = "data-encryption"
	ClusterLabelKey        = "kubermatic.k8c.io/cluster"
	SecretRevisionLabelKey = "kubermatic.k8c.io/secret-revision"
)

type encryptionData interface {
	ImageRegistry(string) string
}

func EncryptionJobCreator(data encryptionData, cluster *kubermaticv1.Cluster, secret *corev1.Secret, key string) batchv1.Job {
	resourceList := strings.Join(cluster.Spec.EncryptionConfiguration.Resources, ",")

	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", EncryptionJobPrefix, cluster.Name),
			Namespace:    cluster.Status.NamespaceName,
			Labels: map[string]string{
				ClusterLabelKey:        cluster.Name,
				SecretRevisionLabelKey: secret.ObjectMeta.ResourceVersion,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:             pointer.Int32(1),
			Completions:             pointer.Int32(1),
			BackoffLimit:            pointer.Int32(0),
			TTLSecondsAfterFinished: pointer.Int32(86400),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "encryption-runner",
							Image:   data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/util:2.0.0",
							Command: []string{"/bin/bash"},
							Args: []string{
								"-c",
								// TODO: this is terribly dangerous and might result in resetting some resources to an older version of it.
								// Replace this with something better!
								fmt.Sprintf("kubectl get %s --all-namespaces -o json | kubectl replace -f -", resourceList),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "KUBECONFIG",
									Value: "/opt/kubeconfig/kubeconfig",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubeconfig",
									ReadOnly:  true,
									MountPath: "/opt/kubeconfig",
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kubeconfig",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: resources.AdminKubeconfigSecretName,
								},
							},
						},
					},
				},
			},
		},
	}
}
