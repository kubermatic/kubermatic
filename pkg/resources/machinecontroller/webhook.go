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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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
		Name: {
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

// WebhookDeploymentCreator returns the function to create and update the machine controller webhook deployment
func WebhookDeploymentCreator(data machinecontrollerData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return resources.MachineControllerWebhookDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Name = resources.MachineControllerWebhookDeploymentName
			dep.Labels = resources.BaseAppLabels(resources.MachineControllerWebhookDeploymentName, nil)
			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.MachineControllerWebhookDeploymentName, nil),
			}
			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

			envVars, err := getEnvVars(data)
			if err != nil {
				return nil, err
			}

			volumes := []corev1.Volume{getKubeconfigVolume(), getServingCertVolume()}
			dep.Spec.Template.Spec.Volumes = volumes
			podLabels, err := data.GetPodTemplateLabels(resources.MachineControllerWebhookDeploymentName, volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create pod labels: %v", err)
			}
			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{Labels: podLabels}

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    Name,
					Image:   data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/machine-controller:" + Tag,
					Command: []string{"/usr/local/bin/webhook"},
					Args: []string{
						"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
						"-logtostderr",
						"-v", "4",
						"-listen-address", "0.0.0.0:9876",
					},
					Env: append(envVars, corev1.EnvVar{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					}),
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
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
						Handler: corev1.Handler{
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
							MountPath: "/etc/kubernetes/kubeconfig",
							ReadOnly:  true,
						},
						{
							Name:      resources.MachineControllerWebhookServingCertSecretName,
							MountPath: "/tmp/cert",
							ReadOnly:  true,
						},
					},
				},
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, webhookResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(Name), "Machine,cluster.k8s.io/v1alpha1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}
	}
}

// ServiceCreator returns the function to reconcile the DNS service
func ServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return resources.MachineControllerWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			se.Name = resources.MachineControllerWebhookServiceName
			se.Labels = resources.BaseAppLabels(resources.MachineControllerWebhookDeploymentName, nil)

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Selector = map[string]string{
				resources.AppLabelKey: resources.MachineControllerWebhookDeploymentName,
			}
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

func getServingCertVolume() corev1.Volume {
	return corev1.Volume{
		Name: resources.MachineControllerWebhookServingCertSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: resources.MachineControllerWebhookServingCertSecretName,
			},
		},
	}
}

type tlsServingCertCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

// TLSServingCertificateCreator returns a function to create/update the secret with the machine-controller-webhook tls certificate
func TLSServingCertificateCreator(data tlsServingCertCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.MachineControllerWebhookServingCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %v", err)
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
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.MachineControllerWebhookServingCertCertKeyName, err)
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
				return nil, fmt.Errorf("failed to generate serving cert: %v", err)
			}
			se.Data[resources.MachineControllerWebhookServingCertCertKeyName] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.MachineControllerWebhookServingCertKeyKeyName] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}
