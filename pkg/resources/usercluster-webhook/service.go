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
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	certutil "k8s.io/client-go/util/cert"
)

// ServiceReconciler returns the function to reconcile the usercluster webhook service.
func ServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return resources.UserClusterWebhookServiceName, func(se *corev1.Service) (*corev1.Service, error) {
			baseLabels := resources.BaseAppLabels(resources.UserClusterWebhookServiceName, nil)
			kubernetes.EnsureLabels(se, baseLabels)

			se.Spec.Type = corev1.ServiceTypeClusterIP
			se.Spec.Selector = resources.BaseAppLabels(resources.UserClusterWebhookDeploymentName, nil)
			se.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "seed",
					Port:       resources.UserClusterWebhookSeedListenPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(seedWebhookListenPort),
				},
				{
					Name:       "user",
					Port:       resources.UserClusterWebhookUserListenPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(userWebhookListenPort),
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

// TLSServingCertificateReconciler returns a function to create/update the secret with the machine-webhook tls certificate.
func TLSServingCertificateReconciler(data tlsServingCertReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.UserClusterWebhookServingCertSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %w", err)
			}
			commonName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.UserClusterWebhookServiceName, data.Cluster().Status.NamespaceName)
			altNames := certutil.AltNames{
				DNSNames: []string{
					resources.UserClusterWebhookServiceName,
					fmt.Sprintf("%s.%s", resources.UserClusterWebhookServiceName, data.Cluster().Status.NamespaceName),
					commonName,
					fmt.Sprintf("%s.%s.svc", resources.UserClusterWebhookServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.", resources.UserClusterWebhookServiceName, data.Cluster().Status.NamespaceName),
				},
			}
			if b, exists := se.Data[resources.ServingCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.ServingCertSecretKey, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}

			newKP, err := triple.NewServerKeyPair(ca,
				commonName,
				resources.UserClusterWebhookServiceName,
				data.Cluster().Status.NamespaceName,
				"",
				nil,
				// For some reason the name the APIServer validates against must be in the SANs, having it as CN is not enough
				[]string{commonName})
			if err != nil {
				return nil, fmt.Errorf("failed to generate serving cert: %w", err)
			}
			se.Data[resources.ServingCertSecretKey] = triple.EncodeCertPEM(newKP.Cert)
			se.Data[resources.ServingCertKeySecretKey] = triple.EncodePrivateKeyPEM(newKP.Key)
			// Include the CA for simplicity
			se.Data[resources.CACertSecretKey] = triple.EncodeCertPEM(ca.Cert)

			return se, nil
		}
	}
}
