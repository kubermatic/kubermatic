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

package vpa

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/utils/pointer"
)

const (
	AdmissionControllerName                  = "vpa-admission-controller"
	AdmissionControllerServiceCertSecretName = "vpa-tls-certs"
	AdmissionControllerServingCertName       = "serverCert.pem"
	AdmissionControllerServingCertKeyName    = "serverKey.pem"
	AdmissionControllerServingCertCAName     = "caCert.pem"

	// WebhookServiceName is defined by the admission-controller's self-registration.
	WebhookServiceName = "vpa-webhook"

	admissionControllerPort = 8944
)

func AdmissionControllerServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return AdmissionControllerName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func AdmissionControllerDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration, versions kubermatic.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return AdmissionControllerName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: appPodLabels(AdmissionControllerName),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(admissionControllerPort),
				"fluentbit.io/parser":  "glog",
			}

			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "tls-certs",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: AdmissionControllerServiceCertSecretName,
						},
					},
				},
			}

			d.Spec.Template.Spec.ServiceAccountName = AdmissionControllerName
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "admission-controller",
					Image:   cfg.Spec.VerticalPodAutoscaler.AdmissionController.DockerRepository + ":" + versions.VPA,
					Command: []string{"/admission-controller"},
					Args: []string{
						fmt.Sprintf("--address=:%d", admissionControllerPort),
						"--logtostderr",
					},
					Env: []corev1.EnvVar{
						{
							Name: "NAMESPACE",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "tls-certs",
							MountPath: "/etc/tls-certs",
							ReadOnly:  true,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "admission",
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "metrics",
							ContainerPort: admissionControllerPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: cfg.Spec.VerticalPodAutoscaler.AdmissionController.Resources,
				},
			}

			return d, nil
		}
	}
}

func AdmissionControllerServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return WebhookServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Ports = []corev1.ServicePort{
				{
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8000),
				},
			}

			s.Spec.Selector = appPodLabels(AdmissionControllerName)

			return s, nil
		}
	}
}

func AdmissionControllerServingCertCreator() reconciling.NamedSecretCreatorGetter {
	altNames := certutil.AltNames{
		DNSNames: []string{
			fmt.Sprintf("%s.%s", WebhookServiceName, metav1.NamespaceSystem),
			fmt.Sprintf("%s.%s.svc", WebhookServiceName, metav1.NamespaceSystem),
		},
	}

	return func() (string, reconciling.SecretCreator) {
		return AdmissionControllerServiceCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			if hasValidCertificate(se, WebhookServiceName, altNames) {
				return se, nil
			}

			ca, err := triple.NewCA(AdmissionControllerName)
			if err != nil {
				return nil, fmt.Errorf("failed to create CA: %v", err)
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a server private key: %v", err)
			}

			config := certutil.Config{
				CommonName: WebhookServiceName,
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[AdmissionControllerServingCertKeyName] = triple.EncodePrivateKeyPEM(key)
			se.Data[AdmissionControllerServingCertName] = triple.EncodeCertPEM(cert)
			se.Data[AdmissionControllerServingCertCAName] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}

func hasValidCertificate(secret *corev1.Secret, commonName string, altNames certutil.AltNames) bool {
	caCert, ok := secret.Data[AdmissionControllerServingCertCAName]
	if !ok {
		return false
	}

	cert, exists := secret.Data[AdmissionControllerServingCertName]
	if !exists {
		return false
	}

	key, exists := secret.Data[AdmissionControllerServingCertKeyName]
	if !exists {
		return false
	}

	// check that the key belongs to the cert
	if _, err := tls.X509KeyPair(cert, key); err != nil {
		return false
	}

	// check that the cert belongs to the CA, covers all the alt names and has not expired
	x509certs, err := certutil.ParseCertsPEM(cert)
	if err != nil {
		return false
	}

	x509caCerts, err := certutil.ParseCertsPEM(caCert)
	if err != nil {
		return false
	}

	return resources.IsServerCertificateValidForAllOf(x509certs[0], commonName, altNames, x509caCerts[0])
}
