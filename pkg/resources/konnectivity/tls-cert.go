/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package konnectivity

import (
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type tlsServingCertCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// TLSServingCertificateCreator returns a function to create/update the secret with the konnectivity proxy server tls certificate used to serve https.
func TLSServingCertificateCreator(data tlsServingCertCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.KonnectivityProxyTLSSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			inClusterIP, err := resources.InClusterApiserverIP(data.Cluster())
			if err != nil {
				return nil, fmt.Errorf("failed to get the in-cluster ClusterIP for the apiserver: %v", err)
			}

			altNames := certutil.AltNames{
				DNSNames: []string{
					fmt.Sprintf("konnectivity-server.%s", data.Cluster().Address.ExternalName),
				},
				IPs: []net.IP{
					*inClusterIP,
					net.ParseIP("127.0.0.1"),
				},
			}

			if b, exists := se.Data[resources.KonnectivityProxyTLSSecretName+".crt"]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.KonnectivityProxyTLSSecretName+".crt", err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], "konnectivity-proxy", altNames, ca.Cert) {
					return se, nil
				}
			}

			if data.Cluster().Spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling {
				externalIP := data.Cluster().Address.IP
				if externalIP == "" {
					return nil, errors.New("externalIP is unset")
				}

				externalIPParsed := net.ParseIP(externalIP)
				if externalIPParsed == nil {
					return nil, errors.New("no external IP")
				}
				altNames.IPs = append(altNames.IPs, externalIPParsed)
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a server private key: %v", err)
			}

			config := certutil.Config{
				CommonName: "konnectivity-proxy",
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[resources.KonnectivityProxyTLSSecretName+".key"] = triple.EncodePrivateKeyPEM(key)
			se.Data[resources.KonnectivityProxyTLSSecretName+".crt"] = triple.EncodeCertPEM(cert)

			return se, nil
		}
	}
}
