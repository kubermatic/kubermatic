/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"crypto/elliptic"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
)

// ResolveKeySpec translates the KeySpec that was frozen into a Cluster when it
// was created into the parameters used by the key generators.
//
// A nil spec, or one that leaves fields empty, resolves to the legacy default
// of RSA-2048. This is deliberately enforced here in code rather than relying
// on defaulting webhooks, so that a cluster which predates the KeyConfiguration
// field keeps its original key material even if it was created by a direct API
// write, or if the CRDs and the webhook are rolled out in an unexpected order.
func ResolveKeySpec(spec *kubermaticv1.KeySpec) (triple.KeyConfig, error) {
	if spec == nil {
		return triple.KeyConfig{}, nil
	}

	switch spec.Algorithm {
	case kubermaticv1.KeyAlgorithmECDSA:
		curve, err := resolveCurve(spec.ECDSACurve)
		if err != nil {
			return triple.KeyConfig{}, err
		}
		return triple.KeyConfig{ECDSACurve: curve}, nil

	case kubermaticv1.KeyAlgorithmRSA, "":
		size := spec.RSAKeySize
		if size == 0 {
			size = kubermaticv1.DefaultRSAKeySize
		}
		return triple.KeyConfig{RSAKeySize: int(size)}, nil

	default:
		// Never silently fall back to a different algorithm than the one that
		// was asked for; an unknown value is a configuration error.
		return triple.KeyConfig{}, fmt.Errorf("unknown key algorithm %q", spec.Algorithm)
	}
}

func resolveCurve(curve kubermaticv1.ECDSACurve) (elliptic.Curve, error) {
	switch curve {
	case kubermaticv1.ECDSACurveP256, "":
		return elliptic.P256(), nil
	case kubermaticv1.ECDSACurveP384:
		return elliptic.P384(), nil
	default:
		return nil, fmt.Errorf("unknown ECDSA curve %q", curve)
	}
}

// ServiceAccountKeyConfig returns the key parameters for the given cluster's
// service-account token signing key.
func ServiceAccountKeyConfig(cluster *kubermaticv1.Cluster) (triple.KeyConfig, error) {
	config, err := ResolveKeySpec(serviceAccountKeySpec(cluster))
	if err != nil {
		return triple.KeyConfig{}, fmt.Errorf("invalid service account key configuration: %w", err)
	}
	return config, nil
}

// CertificateKeyConfig returns the key parameters for the given cluster's
// internal CAs and leaf certificates.
func CertificateKeyConfig(cluster *kubermaticv1.Cluster) (triple.KeyConfig, error) {
	config, err := ResolveKeySpec(certificateKeySpec(cluster))
	if err != nil {
		return triple.KeyConfig{}, fmt.Errorf("invalid certificate key configuration: %w", err)
	}
	return config, nil
}

func serviceAccountKeySpec(cluster *kubermaticv1.Cluster) *kubermaticv1.KeySpec {
	if cluster == nil || cluster.Spec.KeyConfiguration == nil {
		return nil
	}
	return cluster.Spec.KeyConfiguration.ServiceAccountKey
}

func certificateKeySpec(cluster *kubermaticv1.Cluster) *kubermaticv1.KeySpec {
	if cluster == nil || cluster.Spec.KeyConfiguration == nil {
		return nil
	}
	return cluster.Spec.KeyConfiguration.Certificates
}

// ClusterGetter is implemented by the reconcile data structs that carry the
// cluster whose frozen key configuration applies.
type ClusterGetter interface {
	Cluster() *kubermaticv1.Cluster
}

// ClusterCertificateKeyConfigGetter resolves the certificate key parameters of
// the given cluster when the reconciler runs.
func ClusterCertificateKeyConfigGetter(data ClusterGetter) triple.KeyConfigGetter {
	return func() (triple.KeyConfig, error) {
		return CertificateKeyConfig(data.Cluster())
	}
}
