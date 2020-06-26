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

package servingcerthelper

import (
	"crypto/x509"
	"fmt"
	"net"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

// CAGetter is a func to retrieve a CACert and Key
type CAGetter = func() (*triple.KeyPair, error)

// GetServingCertSecretCreator returns a NamedSecretCreatorGetter for a tls serving cert
// using the config options passed in
func ServingCertSecretCreator(caGetter CAGetter, secretName, commonName string, altNamesDNS []string, altNamesIP []net.IP) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return secretName, func(s *corev1.Secret) (*corev1.Secret, error) {

			ca, err := caGetter()
			if err != nil {
				return nil, fmt.Errorf("failed to get CA: %v", err)
			}
			altNames := certutil.AltNames{
				DNSNames: altNamesDNS,
				IPs:      altNamesIP,
			}
			if b, exists := s.Data[resources.ServingCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.ApiserverTLSCertSecretKey, err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return s, nil
				}
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("unable to create a serving cert key: %v", err)
			}

			config := certutil.Config{
				CommonName: commonName,
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign serving certificate: %v", err)
			}

			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			s.Data[resources.ServingCertSecretKey] = triple.EncodeCertPEM(cert)
			s.Data[resources.ServingCertKeySecretKey] = triple.EncodePrivateKeyPEM(key)
			s.Data["tls.crt"] = triple.EncodeCertPEM(cert)
			s.Data["tls.key"] = triple.EncodePrivateKeyPEM(key)

			return s, nil
		}
	}
}
