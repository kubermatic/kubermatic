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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	EncryptionJobPrefix    = "data-encryption"
	ClusterLabelKey        = "kubermatic.k8c.io/cluster"
	SecretRevisionLabelKey = "kubermatic.k8c.io/secret-revision"
	AppLabelValue          = "encryption-runner"

	encryptionJobScript = `
resources=$(kubectl get %s --all-namespaces --output json | jq -r '.items[] | "\(.metadata.namespace // "default"):\(.kind):\(.metadata.name)"');
for res in $resources; do
    ns=$(cut -d':' -f1 <<< $res);
    kind=$(cut -d':' -f2 <<< $res);
    name=$(cut -d':' -f3 <<< $res);
    kubectl get $kind/$name --namespace $ns --output json | kubectl replace --filename -;
done
`
)

type encryptionData interface {
	RewriteImage(string) (string, error)
}

func EncryptionJobCreator(data encryptionData, cluster *kubermaticv1.Cluster, secret *corev1.Secret, res []string, key string) batchv1.Job {
	resourceList := strings.Join(res, ",")

	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", EncryptionJobPrefix, cluster.Name),
			Namespace:    cluster.Status.NamespaceName,
			Labels: map[string]string{
				resources.AppLabelKey:  AppLabelValue,
				ClusterLabelKey:        cluster.Name,
				SecretRevisionLabelKey: secret.ResourceVersion,
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:             ptr.To[int32](1),
			Completions:             ptr.To[int32](1),
			BackoffLimit:            ptr.To[int32](0),
			TTLSecondsAfterFinished: ptr.To[int32](86400),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "encryption-runner",
							Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/util:2.7.0")),
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								fmt.Sprintf(encryptionJobScript, resourceList),
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
