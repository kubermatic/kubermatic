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

package certificates

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CABundleConfigMapReconciler returns a ConfigMapReconcilerFactory that
// creates a ca-bundle ConfigMap for use in seeds and userclusters.
//
// TODO: Do not use fmt.Stringer, but a better type for the CA bundle
//
//	parameter. "*CABundle" is not viable because most of the codebase
//	deals with "resources.CABundle", which in turn exists to
//	prevent an import loop between this and the "resources" package.
func CABundleConfigMapReconciler(name string, caBundle fmt.Stringer) reconciling.NamedConfigMapReconcilerFactory { //nolint:interfacer
	return func() (string, reconciling.ConfigMapReconciler) {
		return name, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			c.Data = map[string]string{
				resources.CABundleConfigMapKey: caBundle.String(),
			}

			return c, nil
		}
	}
}

// CABundle represents an x509.CertPool that was loaded from a file
// and which needs to be access both as a cert pool (i.e. parsed)
// _and_ as a file/PEM string.
type CABundle struct {
	pool     *x509.CertPool
	filename string
	bytes    []byte
}

func NewCABundleFromFile(filename string) (*CABundle, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	bundle, err := NewCABundleFromBytes(bytes)
	if err != nil {
		return nil, err
	}
	bundle.filename = filename

	return bundle, nil
}

func NewCABundleFromBytes(bytes []byte) (*CABundle, error) {
	if err := ValidateCABundle(string(bytes)); err != nil {
		return nil, fmt.Errorf("CA bundle is invalid: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(bytes) {
		return nil, errors.New("CA file does not contain any valid PEM-encoded certificates")
	}

	return &CABundle{
		pool:  pool,
		bytes: bytes,
	}, nil
}

// NewFakeCABundle returns a CA bundle that contains a single certificate
// that cannot validate anything.
func NewFakeCABundle() *CABundle {
	caBundle, _ := NewCABundleFromBytes([]byte(`-----BEGIN CERTIFICATE-----
MIIDdTCCAl2gAwIBAgILBAAAAAABFUtaw5QwDQYJKoZIhvcNAQEFBQAwVzELMAkGA1UEBhMCQkUx
GTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsTB1Jvb3QgQ0ExGzAZBgNVBAMTEkds
b2JhbFNpZ24gUm9vdCBDQTAeFw05ODA5MDExMjAwMDBaFw0yODAxMjgxMjAwMDBaMFcxCzAJBgNV
BAYTAkJFMRkwFwYDVQQKExBHbG9iYWxTaWduIG52LXNhMRAwDgYDVQQLEwdSb290IENBMRswGQYD
VQQDExJHbG9iYWxTaWduIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDa
DuaZjc6j40+Kfvvxi4Mla+pIH/EqsLmVEQS98GPR4mdmzxzdzxtIK+6NiY6arymAZavpxy0Sy6sc
THAHoT0KMM0VjU/43dSMUBUc71DuxC73/OlS8pF94G3VNTCOXkNz8kHp1Wrjsok6Vjk4bwY8iGlb
Kk3Fp1S4bInMm/k8yuX9ifUSPJJ4ltbcdG6TRGHRjcdGsnUOhugZitVtbNV4FpWi6cgKOOvyJBNP
c1STE4U6G7weNLWLBYy5d4ux2x8gkasJU26Qzns3dLlwR5EiUWMWea6xrkEmCMgZK9FGqkjWZCrX
gzT/LCrBbBlDSgeF59N89iFo7+ryUp9/k5DPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNV
HRMBAf8EBTADAQH/MB0GA1UdDgQWBBRge2YaRQ2XyolQL30EzTSo//z9SzANBgkqhkiG9w0BAQUF
AAOCAQEA1nPnfE920I2/7LqivjTFKDK1fPxsnCwrvQmeU79rXqoRSLblCKOzyj1hTdNGCbM+w6Dj
Y1Ub8rrvrTnhQ7k4o+YviiY776BQVvnGCv04zcQLcFGUl5gE38NflNUVyRRBnMRddWQVDf9VMOyG
j/8N7yy5Y0b2qvzfvGn9LhJIZJrglfCm7ymPAbEVtQwdpf5pLGkkeB6zpxxxYu7KyJesF12KwvhH
hm4qxFYxldBniYUr+WymXUadDKqC5JlR3XC321Y9YeRq4VzW9v493kHMB65jUr9TU/Qr6cf9tveC
X4XSQRjbgbMEHMUfpIBvFSDJ3gyICh3WZlXi/EjJKSZp4A==
-----END CERTIFICATE-----`))

	return caBundle
}

func (b *CABundle) CertPool() *x509.CertPool {
	return b.pool
}

func (b *CABundle) String() string {
	return string(b.bytes)
}

func GlobalCABundle(ctx context.Context, client ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration) (*corev1.ConfigMap, error) {
	caBundle := &corev1.ConfigMap{}
	key := types.NamespacedName{Name: config.Spec.CABundle.Name, Namespace: config.Namespace}

	if err := client.Get(ctx, key, caBundle); err != nil {
		return nil, fmt.Errorf("failed to fetch CA bundle: %w", err)
	}

	if err := ValidateCABundleConfigMap(caBundle); err != nil {
		return nil, fmt.Errorf("CA bundle is invalid: %w", err)
	}

	return caBundle, nil
}

func ValidateCABundleConfigMap(cm *corev1.ConfigMap) error {
	bundle, ok := cm.Data[resources.CABundleConfigMapKey]
	if !ok {
		return fmt.Errorf("ConfigMap does not contain key %q", resources.CABundleConfigMapKey)
	}

	return ValidateCABundle(bundle)
}

func ValidateCABundle(bundle string) error {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(bundle)) {
		return errors.New("bundle does not contain any valid certificates")
	}

	return nil
}
