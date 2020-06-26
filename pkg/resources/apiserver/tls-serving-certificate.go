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

package apiserver

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type tlsServingCertCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// TLSServingCertificateCreator returns a function to create/update the secret with the apiserver tls certificate used to serve https
func TLSServingCertificateCreator(data tlsServingCertCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.ApiserverTLSSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			externalIP := data.Cluster().Address.IP
			if externalIP == "" {
				return nil, errors.New("externalIP is unset")
			}

			inClusterIP, err := resources.InClusterApiserverIP(data.Cluster())
			if err != nil {
				return nil, fmt.Errorf("failed to get the in-cluster ClusterIP for the apiserver: %v", err)
			}
			externalIPParsed := net.ParseIP(externalIP)
			if externalIPParsed == nil {
				return nil, errors.New("no external IP")
			}

			altNames := certutil.AltNames{
				DNSNames: []string{
					// ExternalName
					data.Cluster().Address.ExternalName,
					// User facing
					"kubernetes",
					"kubernetes.default",
					"kubernetes.default.svc",
					fmt.Sprintf("kubernetes.default.svc.%s", data.Cluster().Spec.ClusterNetwork.DNSDomain),
					// Internal - apiserver-external
					resources.ApiserverExternalServiceName,
					fmt.Sprintf("%s.%s", resources.ApiserverExternalServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc", resources.ApiserverExternalServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.cluster.local", resources.ApiserverExternalServiceName, data.Cluster().Status.NamespaceName),
					// Internal - apiserver
					resources.ApiserverInternalServiceName,
					fmt.Sprintf("%s.%s", resources.ApiserverInternalServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc", resources.ApiserverInternalServiceName, data.Cluster().Status.NamespaceName),
					fmt.Sprintf("%s.%s.svc.cluster.local", resources.ApiserverInternalServiceName, data.Cluster().Status.NamespaceName),
				},
				IPs: []net.IP{
					externalIPParsed,
					*inClusterIP,
					net.ParseIP("127.0.0.1"),
				},
			}

			if b, exists := se.Data[resources.ApiserverTLSCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.ApiserverTLSCertSecretKey, err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], "kube-apiserver", altNames, ca.Cert) {
					return se, nil
				}
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a server private key: %v", err)
			}

			config := certutil.Config{
				CommonName: "kube-apiserver",
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[resources.ApiserverTLSKeySecretKey] = triple.EncodePrivateKeyPEM(key)
			se.Data[resources.ApiserverTLSCertSecretKey] = triple.EncodeCertPEM(cert)

			return se, nil
		}
	}
}
