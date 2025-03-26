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

package etcd

import (
	"crypto/x509"
	"fmt"
	"net"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	certutil "k8s.io/client-go/util/cert"
)

type tlsCertificateReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// TLSCertificateReconciler returns a function to create/update the secret with the etcd tls certificate.
func TLSCertificateReconciler(data tlsCertificateReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.EtcdTLSCertificateSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %w", err)
			}

			altNames := certutil.AltNames{
				DNSNames: []string{
					"localhost",
				},
				IPs: []net.IP{
					net.ParseIP("127.0.0.1"),
				},
			}
			etcdClusterSize := kubermaticv1.DefaultEtcdClusterSize
			if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
				etcdClusterSize = kubermaticv1.MaxEtcdClusterSize
			}
			for i := range etcdClusterSize {
				// Member name
				podName := fmt.Sprintf("etcd-%d", i)
				altNames.DNSNames = append(altNames.DNSNames, podName)

				// Pod DNS name
				absolutePodDNSName := fmt.Sprintf("etcd-%d.%s.%s.svc.cluster.local", i, resources.EtcdServiceName, data.Cluster().Status.NamespaceName)
				altNames.DNSNames = append(altNames.DNSNames, absolutePodDNSName)
			}

			if b, exists := se.Data[resources.EtcdTLSCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret %s: %w", resources.EtcdTLSCertSecretKey, resources.EtcdTLSCertificateSecretName, err)
				}

				if resources.IsServerCertificateValidForAllOf(certs[0], "etcd", altNames, ca.Cert) {
					return se, nil
				}
			}

			key, err := triple.NewPrivateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to create private key for etcd server tls certificate: %w", err)
			}

			config := certutil.Config{
				CommonName: "etcd",
				AltNames:   altNames,
				Usages: []x509.ExtKeyUsage{
					x509.ExtKeyUsageServerAuth,
					// etcd needs even the server cert to allow client authentication, see
					// https://github.com/openshift/kubecsr/commit/aad75d333646da657e788db93f2ff75da850b542
					// not having it does not cause etcd to fail, but produces lots of warnings and errors
					x509.ExtKeyUsageClientAuth,
				},
			}

			cert, err := triple.NewSignedCert(config, key, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %w", err)
			}

			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data[resources.EtcdTLSKeySecretKey] = triple.EncodePrivateKeyPEM(key)
			se.Data[resources.EtcdTLSCertSecretKey] = triple.EncodeCertPEM(cert)

			return se, nil
		}
	}
}
