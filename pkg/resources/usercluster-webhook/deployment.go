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

package webhook

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	seedWebhookListenPort = 9443
	userWebhookListenPort = 19443
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

type webhookData interface {
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
	DC() *kubermaticv1.Datacenter
	KubermaticAPIImage() string
	KubermaticDockerTag() string
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	GetEnvVars() ([]corev1.EnvVar, error)
}

// DeploymentReconciler returns the function to create and update the user cluster webhook deployment.
func DeploymentReconciler(data webhookData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.UserClusterWebhookDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(resources.UserClusterWebhookDeploymentName, nil)
			kubernetes.EnsureLabels(d, baseLabels)

			d.Spec.Replicas = ptr.To[int32](1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			kubernetes.EnsureLabels(&d.Spec.Template, baseLabels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape":                 "true",
				"prometheus.io/port":                   "8080",
				"fluentbit.io/parser":                  "json_iso",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			projectID, ok := data.Cluster().Labels[kubermaticv1.ProjectIDLabelKey]
			if !ok {
				return nil, fmt.Errorf("no project-id label on cluster %q", data.Cluster().Name)
			}

			args := []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				fmt.Sprintf("-seed-webhook-listen-port=%d", seedWebhookListenPort),
				"-seed-webhook-cert-dir=/opt/webhook-serving-cert/",
				fmt.Sprintf("-seed-webhook-cert-name=%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-seed-webhook-key-name=%s", resources.ServingCertKeySecretKey),
				fmt.Sprintf("-user-webhook-listen-port=%d", userWebhookListenPort),
				"-user-webhook-cert-dir=/opt/webhook-serving-cert/",
				fmt.Sprintf("-user-webhook-cert-name=%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-user-webhook-key-name=%s", resources.ServingCertKeySecretKey),
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-project-id=%s", projectID),
				fmt.Sprintf("-cluster-name=%s", data.Cluster().Name),
			}

			if data.Cluster().Spec.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			envVars, err := data.GetEnvVars()
			if err != nil {
				return nil, err
			}

			volumes := []corev1.Volume{
				{
					Name: "webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.UserClusterWebhookServingCertSecretName,
						},
					},
				},
				{
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CABundleConfigMapName,
							},
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      "webhook-serving-cert",
					MountPath: "/opt/webhook-serving-cert/",
					ReadOnly:  true,
				},
				{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
					ReadOnly:  true,
				},
				{
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
					ReadOnly:  true,
				},
			}

			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: resources.Bool(true),
				RunAsUser:    resources.Int64(65534),
				RunAsGroup:   resources.Int64(65534),
				FSGroup:      resources.Int64(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.UserClusterControllerDeploymentName,
					Image:   data.KubermaticAPIImage() + ":" + data.KubermaticDockerTag(),
					Command: []string{"user-cluster-webhook"},
					Args:    args,
					Env:     envVars,
					Ports: []corev1.ContainerPort{
						{
							Name:          "admission",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources:    defaultResourceRequirements,
					ReadinessProbe: &corev1.Probe{
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
						ProbeHandler: corev1.ProbeHandler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.Parse("metrics"),
							},
						},
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: resources.Bool(false),
						ReadOnlyRootFilesystem:   resources.Bool(true),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								corev1.Capability("ALL"),
							},
						},
					},
				},
			}

			d.Spec.Template, err = apiserver.IsRunningWrapper(data, d.Spec.Template, sets.New(resources.UserClusterControllerDeploymentName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return d, nil
		}
	}
}
