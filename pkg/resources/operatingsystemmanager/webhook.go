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

package operatingsystemmanager

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	certutil "k8s.io/client-go/util/cert"
)

var (
	webhookResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.OperatingSystemManagerContainerName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("10m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
)

// WebhookDeploymentReconciler returns the function to create and update the operating-system-manager webhook deployment.
func WebhookDeploymentReconciler(data operatingSystemManagerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.OperatingSystemManagerWebhookDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			var err error

			baseLabels := resources.BaseAppLabels(resources.OperatingSystemManagerWebhookDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

			repository := Repository
			tag := Tag

			dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: resources.Bool(true),
				RunAsUser:    resources.Int64(65534),
				RunAsGroup:   resources.Int64(65534),
				FSGroup:      resources.Int64(65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    resources.OperatingSystemManagerContainerName,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/webhook"},
					Args: []string{
						"-log-debug=false",
						"-log-format", "json",
						"-namespace", "kube-system",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "KUBECONFIG",
							Value: "/etc/kubernetes/worker-kubeconfig/kubeconfig",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "webhook-server",
							ContainerPort: 9443,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/readyz",
								Port:   intstr.FromInt(8081),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 8,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8081),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.OperatingSystemManagerWebhookKubeconfigSecretName,
							MountPath: "/etc/kubernetes/worker-kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.OperatingSystemManagerWebhookServingCertSecretName,
							MountPath: "/tmp/k8s-webhook-server/serving-certs",
							ReadOnly:  true,
						},
						{
							Name:      resources.CABundleConfigMapName,
							MountPath: "/etc/kubernetes/pki/ca-bundle",
							ReadOnly:  true,
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

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				getWebhookKubeconfigVolume(),
				getServingCertVolume(),
				getCABundleVolume(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, webhookResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(resources.OperatingSystemManagerContainerName))
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
			}

			return dep, nil
		}
	}
}

// ServiceReconciler returns the function to reconcile the DNS service.
func ServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.OperatingSystemManagerWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			baseLabels := resources.BaseAppLabels(resources.OperatingSystemManagerWebhookDeploymentName, nil)
			kubernetes.EnsureLabels(se, baseLabels)

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Selector = baseLabels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "webhook-server",
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(9443),
				},
			}

			return se, nil
		}
	}
}

type tlsServingCertReconcilerData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

// TLSServingCertificateReconciler returns a function to create/update the secret with the operating-system-manager-webhook tls certificate.
func TLSServingCertificateReconciler(data tlsServingCertReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.OperatingSystemManagerWebhookServingCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %w", err)
			}

			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.OperatingSystemManagerWebhookServiceName, data.Cluster().Status.NamespaceName)
			altNames := certutil.AltNames{
				DNSNames: []string{
					resources.OperatingSystemManagerWebhookServiceName,
					fmt.Sprintf("%s.%s", resources.OperatingSystemManagerWebhookServiceName, data.Cluster().Status.NamespaceName),
					commonName,
					fmt.Sprintf("%s.%s.svc", resources.OperatingSystemManagerWebhookServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.", resources.OperatingSystemManagerWebhookServiceName, data.Cluster().Status.NamespaceName),
				},
			}

			if b, exists := se.Data[resources.OperatingSystemManagerWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.OperatingSystemManagerWebhookServingCertCertKeyName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				resources.OperatingSystemManagerWebhookServiceName,
				data.Cluster().Status.NamespaceName,
				"",
				nil,
				// For some reason the name the APIServer validates against must be in the SANs, having it as CN is not enough
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}

			se.Data[resources.OperatingSystemManagerWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.OperatingSystemManagerWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}

func getServingCertVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.OperatingSystemManagerWebhookServingCertSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.OperatingSystemManagerWebhookServingCertSecretName,
			},
		},
	}
}

func getWebhookKubeconfigVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.OperatingSystemManagerWebhookKubeconfigSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.OperatingSystemManagerWebhookKubeconfigSecretName,
			},
		},
	}
}
