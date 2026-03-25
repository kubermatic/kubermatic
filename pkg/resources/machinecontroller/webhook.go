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

package machinecontroller

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
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
		resources.MachineControllerContainerName: {
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

// WebhookDeploymentReconciler returns the function to create and update the machine controller webhook deployment.
func WebhookDeploymentReconciler(data machinecontrollerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.MachineControllerWebhookDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			args := []string{
				"-worker-cluster-kubeconfig", "/etc/kubernetes/worker-kubeconfig/kubeconfig",
				"-listen-address", "0.0.0.0:9876",
				"-namespace", data.Cluster().Status.NamespaceName,
				"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
				"-tls-cert-path", "/etc/kubernetes/pki/serving-cert/cert.pem",
				"-tls-key-path", "/etc/kubernetes/pki/serving-cert/key.pem",
			}

			externalCloudProvider := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]
			if externalCloudProvider {
				args = append(args, "-node-external-cloud-provider")
			}

			featureGates := data.GetCSIMigrationFeatureGates(nil)
			if len(featureGates) > 0 {
				args = append(args, "-node-kubelet-feature-gates", strings.Join(featureGates, ","))
			}

			baseLabels := resources.BaseAppLabels(resources.MachineControllerWebhookDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			envVars, err := data.GetEnvVars()
			if err != nil {
				return nil, err
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
			})

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

			repository := registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/machine-controller"))
			if r := data.MachineControllerImageRepository(); r != "" {
				repository = r
			}
			tag := Tag
			if t := data.MachineControllerImageTag(); t != "" {
				tag = t
			}

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
					Name:    resources.MachineControllerContainerName,
					Image:   repository + ":" + tag,
					Command: []string{"/usr/local/bin/webhook"},
					Args:    args,
					Env: append(envVars, corev1.EnvVar{
						Name:  "PROBER_KUBECONFIG",
						Value: "/etc/kubernetes/worker-kubeconfig/kubeconfig",
					}),
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(9876),
								Scheme: corev1.URISchemeHTTPS,
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
								Port:   intstr.FromInt(9876),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      15,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.MachineControllerKubeconfigSecretName,
							MountPath: "/etc/kubernetes/worker-kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.MachineControllerWebhookServingCertSecretName,
							MountPath: "/etc/kubernetes/pki/serving-cert",
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

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				getKubeconfigVolume(),
				getServingCertVolume(),
				getCABundleVolume(),
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, webhookResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = webhookServiceAccountName

			dep.Spec.Template, err = apiserver.IsRunningWrapper(data, dep.Spec.Template, sets.New(resources.MachineControllerContainerName), "Machine,cluster.k8s.io/v1alpha1")
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
		return resources.MachineControllerWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			baseLabels := resources.BaseAppLabels(resources.MachineControllerWebhookDeploymentName, nil)
			kubernetes.EnsureLabels(se, baseLabels)

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Selector = baseLabels
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "",
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(9876),
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

// TLSServingCertificateReconciler returns a function to create/update the secret with the machine-controller-webhook tls certificate.
func TLSServingCertificateReconciler(data tlsServingCertReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.MachineControllerWebhookServingCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %w", err)
			}
			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.MachineControllerWebhookServiceName, data.Cluster().Status.NamespaceName)
			altNames := certutil.AltNames{
				DNSNames: []string{
					resources.MachineControllerWebhookServiceName,
					fmt.Sprintf("%s.%s", resources.MachineControllerWebhookServiceName, data.Cluster().Status.NamespaceName),
					commonName,
					fmt.Sprintf("%s.%s.svc", resources.MachineControllerWebhookServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.", resources.MachineControllerWebhookServiceName, data.Cluster().Status.NamespaceName),
				},
			}
			if b, exists := se.Data[resources.MachineControllerWebhookServingCertCertKeyName]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.MachineControllerWebhookServingCertCertKeyName, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				resources.MachineControllerWebhookServiceName,
				data.Cluster().Status.NamespaceName,
				"",
				nil,
				// For some reason the name the APIServer validates against must be in the SANs, having it as CN is not enough
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}
			se.Data[resources.MachineControllerWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.MachineControllerWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}
