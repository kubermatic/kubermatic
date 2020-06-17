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

package openvpn

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type tlsServingCertCreatorData interface {
	GetOpenVPNCA() (*resources.ECDSAKeyPair, error)
}

// TLSServingCertificateCreator returns a function to create/update a secret with the openvpn server tls certificate
func TLSServingCertificateCreator(data tlsServingCertCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.OpenVPNServerCertificatesSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetOpenVPNCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get openvpn ca: %v", err)
			}
			altNames := certutil.AltNames{}
			if b, exists := se.Data[resources.OpenVPNServerCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.OpenVPNServerCertSecretKey, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], "openvpn-server", altNames, ca.Cert) {
					return se, nil
				}
			}
			config := certutil.Config{
				CommonName: "openvpn-server",
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}
			cert, key, err := certificates.GetSignedECDSACertAndKey(certificates.Duration365d, config, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[resources.OpenVPNServerCertSecretKey] = cert
			se.Data[resources.OpenVPNServerKeySecretKey] = key

			return se, nil
		}
	}
}

// CACreator returns a function to create the ECDSA-based CA to be used for OpenVPN
func CACreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.OpenVPNCASecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			if data, exists := se.Data[resources.OpenVPNCACertKey]; exists {
				certs, err := certutil.ParseCertsPEM(data)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate %s from existing secret %s: %v",
						resources.OpenVPNCACertKey, resources.OpenVPNCASecretName, err)
				}
				if !resources.CertWillExpireSoon(certs[0]) {
					return se, nil
				}
			}

			cert, key, err := certificates.GetECDSACACertAndKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate OpenVPN CA: %v", err)
			}
			se.Data[resources.OpenVPNCACertKey] = cert
			se.Data[resources.OpenVPNCAKeyKey] = key

			return se, nil
		}
	}
}
