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

package v1

import "testing"

func TestKubermaticConfigurationDeepCopyGatewayInfrastructureAnnotations(t *testing.T) {
	original := &KubermaticConfiguration{
		Spec: KubermaticConfigurationSpec{
			Ingress: KubermaticIngressConfiguration{
				Domain: "example.com",
				Gateway: &KubermaticGatewayConfiguration{
					InfrastructureAnnotations: map[string]string{
						"metallb.io/address-pool": "public",
					},
				},
			},
		},
	}

	copied := original.DeepCopy()
	copied.Spec.Ingress.Gateway.InfrastructureAnnotations["metallb.io/address-pool"] = "private"
	copied.Spec.Ingress.Gateway.InfrastructureAnnotations["example.com/custom"] = "value"

	if got := original.Spec.Ingress.Gateway.InfrastructureAnnotations["metallb.io/address-pool"]; got != "public" {
		t.Fatalf("expected original annotation to remain unchanged, got %q", got)
	}

	if got := original.Spec.Ingress.Gateway.InfrastructureAnnotations["example.com/custom"]; got != "" {
		t.Fatalf("expected original annotations to stay isolated from the copy, got %q", got)
	}
}

func TestKubermaticConfigurationDeepCopyCopiesGatewayTLS(t *testing.T) {
	cfg := &KubermaticConfiguration{
		Spec: KubermaticConfigurationSpec{
			Ingress: KubermaticIngressConfiguration{
				Domain: "example.com",
				Gateway: &KubermaticGatewayConfiguration{
					TLS: &KubermaticGatewayTLSConfiguration{
						SecretRef: &KubermaticGatewaySecretReference{
							Name:      "manual-wildcard",
							Namespace: "shared-certs",
						},
					},
				},
			},
		},
	}

	cloned := cfg.DeepCopy()
	cloned.Spec.Ingress.Gateway.TLS.SecretRef.Name = "other-secret"
	cloned.Spec.Ingress.Gateway.TLS.SecretRef.Namespace = "other-namespace"

	if cfg.Spec.Ingress.Gateway.TLS.SecretRef.Name != "manual-wildcard" {
		t.Fatalf("expected original secret name to remain unchanged, got %q", cfg.Spec.Ingress.Gateway.TLS.SecretRef.Name)
	}

	if cfg.Spec.Ingress.Gateway.TLS.SecretRef.Namespace != "shared-certs" {
		t.Fatalf("expected original secret namespace to remain unchanged, got %q", cfg.Spec.Ingress.Gateway.TLS.SecretRef.Namespace)
	}
}
