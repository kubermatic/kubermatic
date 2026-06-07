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

package validation

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestValidateKubermaticConfigurationVersions(t *testing.T) {
	testcases := []struct {
		name           string
		versions       []string
		defaultVersion string
		updates        []kubermaticv1.Update
		valid          bool
	}{
		{
			name:           "vanilla, single version",
			versions:       []string{"v1.10.5"},
			defaultVersion: "v1.10.5",
			valid:          true,
		},
		{
			name:           "regular version update",
			versions:       []string{"v1.10.5", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          true,
		},
		{
			name:           "order does not matter",
			versions:       []string{"v1.11.4", "v1.12.3", "1.9.2", "v1.10.8"},
			defaultVersion: "v1.10.8",
			valid:          true,
		},
		{
			name:           "missing v1.12",
			versions:       []string{"v1.11.4", "v1.13.3"},
			defaultVersion: "v1.13.3",
			valid:          false,
		},
		{
			name:           "order does not matter",
			versions:       []string{"v1.13.3", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          false,
		},
		{
			name:           "large gaps are also detected",
			versions:       []string{"v1.15.4", "v1.11.4"},
			defaultVersion: "v1.11.4",
			valid:          false,
		},
		{
			name:           "no default configured",
			versions:       []string{"v1.15.4", "v1.11.4"},
			defaultVersion: "",
			valid:          false,
		},
		{
			name:           "invalid default configured",
			versions:       []string{"v1.2.2", "v1.2.4"},
			defaultVersion: "v1.2.3",
			valid:          false,
		},
		{
			name:           "should allow updates with automatic update rules from concrete version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.1",
				To:        "v1.12.2",
				Automatic: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic node update rules from concrete version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.1",
				To:                  "v1.12.2",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic update rules from wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.*",
				To:        "v1.12.2",
				Automatic: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should allow updates with automatic node update rules from wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.*",
				To:                  "v1.12.2",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: true,
		},
		{
			name:           "should forbid updates with automatic update rules to wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:      "v1.11.0",
				To:        "v1.12.*",
				Automatic: ptr.To(true),
			}},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic node update rules to wildcard version",
			versions:       []string{"v1.11.1", "v1.12.2"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{{
				From:                "v1.11.0",
				To:                  "v1.12.*",
				AutomaticNodeUpdate: ptr.To(true),
			}},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic update rules to version with concrete automatic update rule",
			versions:       []string{"v1.11.1", "v1.12.2", "v1.13.3"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{
				{
					From:      "v1.11.1",
					To:        "v1.12.2",
					Automatic: ptr.To(true),
				},
				{
					From:      "v1.12.2",
					To:        "v1.13.3",
					Automatic: ptr.To(true),
				},
			},
			valid: false,
		},
		{
			name:           "should forbid updates with automatic node update rules to version with concrete automatic update rule",
			versions:       []string{"v1.11.1", "v1.12.2", "v1.13.3"},
			defaultVersion: "v1.11.1",
			updates: []kubermaticv1.Update{
				{
					From:                "v1.11.1",
					To:                  "v1.12.2",
					AutomaticNodeUpdate: ptr.To(true),
				},
				{
					From:                "v1.12.2",
					To:                  "v1.13.3",
					AutomaticNodeUpdate: ptr.To(true),
				},
			},
			valid: false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			config := kubermaticv1.KubermaticVersioningConfiguration{}
			if tt.defaultVersion != "" {
				config.Default = semver.NewSemverOrDie(tt.defaultVersion)
			}

			for _, v := range tt.versions {
				version := semver.NewSemverOrDie(v)
				config.Versions = append(config.Versions, *version)
			}

			config.Updates = tt.updates

			errs := ValidateKubermaticVersioningConfiguration(config, nil)
			if tt.valid {
				if len(errs) > 0 {
					t.Fatalf("Expected configuration to be valid, but got err: %v", errs.ToAggregate())
				}
			} else {
				if len(errs) == 0 {
					t.Fatal("Expected configuration to be invalid, but it was accepted.")
				}
			}
		})
	}
}

func TestValidateMirrorImages(t *testing.T) {
	testcases := []struct {
		name         string
		mirrorImages []string
		valid        bool
	}{
		{
			name:         "valid single image",
			mirrorImages: []string{"nginx:1.21.6"},
			valid:        true,
		},
		{
			name:         "valid multiple images",
			mirrorImages: []string{"nginx:1.21.6", "quay.io/kubermatic/kubelb-manager-ee:v1.1.0"},
			valid:        true,
		},
		{
			name:         "invalid image format (missing tag)",
			mirrorImages: []string{"nginx"},
			valid:        false,
		},
		{
			name:         "invalid image format (missing repository)",
			mirrorImages: []string{":latest"},
			valid:        false,
		},
		{
			name:         "invalid image format (empty string)",
			mirrorImages: []string{""},
			valid:        false,
		},
		{
			name:         "invalid image format (extra colon)",
			mirrorImages: []string{"nginx:1.21.6:extra"},
			valid:        false,
		},
		{
			name:         "mixed valid and invalid images",
			mirrorImages: []string{"nginx:1.21.6", "invalid-image"},
			valid:        false,
		},
		{
			name:         "empty mirrorImages list",
			mirrorImages: []string{},
			valid:        true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			spec := newValidKubermaticConfigurationSpec()
			spec.MirrorImages = tt.mirrorImages
			errs := ValidateKubermaticConfigurationSpec(spec)
			if tt.valid {
				if len(errs) > 0 {
					t.Fatalf("Expected configuration to be valid, but got errors: %v", errs.ToAggregate())
				}
			} else {
				if len(errs) == 0 {
					t.Fatal("Expected configuration to be invalid, but it was accepted.")
				}
			}
		})
	}
}

func newValidKubermaticConfigurationSpec() *kubermaticv1.KubermaticConfigurationSpec {
	spec := &kubermaticv1.KubermaticConfigurationSpec{
		Ingress: kubermaticv1.KubermaticIngressConfiguration{
			Domain: "example.com",
		},
	}

	version := semver.NewSemverOrDie("v1.11.1")
	spec.Versions.Default = version
	spec.Versions.Versions = append(spec.Versions.Versions, *version)

	return spec
}

func TestValidateGatewayTLSConfiguration(t *testing.T) {
	baseVersion := semver.NewSemverOrDie("v1.11.1")

	testcases := []struct {
		name  string
		spec  *kubermaticv1.KubermaticConfigurationSpec
		valid bool
	}{
		{
			name: "gateway tls secretRef with name is valid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{
							SecretRef: &kubermaticv1.KubermaticGatewaySecretReference{
								Name:      "kubermatic-tls",
								Namespace: "shared-certs",
							},
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "gateway tls secretRef without name is invalid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{
							SecretRef: &kubermaticv1.KubermaticGatewaySecretReference{},
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "gateway tls secretRef and certificate issuer are mutually exclusive",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					CertificateIssuer: corev1.TypedLocalObjectReference{
						Name: "letsencrypt-prod",
						Kind: "ClusterIssuer",
					},
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{
							SecretRef: &kubermaticv1.KubermaticGatewaySecretReference{
								Name: "kubermatic-tls",
							},
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway rejects leftover tls",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					CertificateIssuer: corev1.TypedLocalObjectReference{
						Name: "letsencrypt-prod",
						Kind: "ClusterIssuer",
					},
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{
							SecretRef: &kubermaticv1.KubermaticGatewaySecretReference{},
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway rejects empty tls",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway rejects custom class name",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						ClassName: "my-custom-gatewayclass",
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway accepts legacy default class name from removed CRD default",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						ClassName: defaulting.DefaultGatewayClassName,
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway rejects infrastructure annotations",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						InfrastructureAnnotations: map[string]string{
							"metallb.io/address-pool": "public",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway allows empty infrastructure annotations",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
						InfrastructureAnnotations: map[string]string{},
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway rejects certificate issuer",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					CertificateIssuer: corev1.TypedLocalObjectReference{
						Name: "letsencrypt-prod",
						Kind: "ClusterIssuer",
					},
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "networking",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway without name is invalid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Namespace: "networking",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "external gateway using default key with omitted namespace is statically valid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name: "kubermatic",
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway name with dots is valid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform.gateway",
							Namespace: "networking",
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway using default key with explicit namespace is statically valid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "kubermatic",
							Namespace: "kubermatic",
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway using default name in different namespace is valid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "kubermatic",
							Namespace: "networking",
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "external gateway with invalid namespace is invalid",
			spec: &kubermaticv1.KubermaticConfigurationSpec{
				Ingress: kubermaticv1.KubermaticIngressConfiguration{
					Domain: "example.com",
					Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
						ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
							Name:      "platform-gateway",
							Namespace: "Not_Valid",
						},
					},
				},
			},
			valid: false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			tt.spec.Versions.Default = baseVersion
			tt.spec.Versions.Versions = append(tt.spec.Versions.Versions, *baseVersion)

			errs := ValidateKubermaticConfigurationSpec(tt.spec)
			if tt.valid {
				if len(errs) > 0 {
					t.Fatalf("Expected configuration to be valid, but got errors: %v", errs.ToAggregate())
				}
			} else {
				if len(errs) == 0 {
					t.Fatal("Expected configuration to be invalid, but it was accepted.")
				}
			}
		})
	}
}

func TestValidateExternalGatewayConfigurationForbidsManagedGatewayFields(t *testing.T) {
	spec := &kubermaticv1.KubermaticConfigurationSpec{
		Ingress: kubermaticv1.KubermaticIngressConfiguration{
			Domain: "example.com",
			CertificateIssuer: corev1.TypedLocalObjectReference{
				Name: "letsencrypt-prod",
				Kind: "ClusterIssuer",
			},
			Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
				ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
					Name:      "platform-gateway",
					Namespace: "networking",
				},
				ClassName: "my-custom-gatewayclass",
				InfrastructureAnnotations: map[string]string{
					"metallb.io/address-pool": "public",
				},
				TLS: &kubermaticv1.KubermaticGatewayTLSConfiguration{
					SecretRef: &kubermaticv1.KubermaticGatewaySecretReference{
						Name: "kubermatic-tls",
					},
				},
			},
		},
	}

	errs := ValidateExternalGatewayConfiguration(spec)
	wantForbiddenPaths := map[string]bool{
		"spec.ingress.gateway.className":                 false,
		"spec.ingress.gateway.infrastructureAnnotations": false,
		"spec.ingress.gateway.tls":                       false,
		"spec.ingress.certificateIssuer":                 false,
	}

	for _, err := range errs {
		if err.Type != field.ErrorTypeForbidden {
			continue
		}
		if _, ok := wantForbiddenPaths[err.Field]; ok {
			wantForbiddenPaths[err.Field] = true
		}
	}

	for path, found := range wantForbiddenPaths {
		if !found {
			t.Fatalf("expected forbidden error for %s, got %v", path, errs.ToAggregate())
		}
	}
}
