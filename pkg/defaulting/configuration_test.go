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

package defaulting_test

import (
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/validation"
)

func TestDefaultConfigurationIsValid(t *testing.T) {
	errs := validation.ValidateKubermaticVersioningConfiguration(defaulting.DefaultKubernetesVersioning, nil)
	for _, err := range errs {
		t.Error(err)
	}
}

func TestDefaultConfigurationClearsLegacyGatewayClassNameWhenExternalGatewayIsSet(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ClassName: defaulting.DefaultGatewayClassName,
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("DefaultConfiguration returned error: %v", err)
	}

	if defaulted.Spec.Ingress.Gateway.ClassName != "" {
		t.Fatalf("expected legacy default ClassName to be cleared, got %q", defaulted.Spec.Ingress.Gateway.ClassName)
	}
}

// TestDefaultConfigurationEnablesHTTPRouteGatewaySyncByDefault covers the 2.31
// enforcement: since Gateway API is the default, cert-manager needs the
// HTTPRouteGatewaySync feature gate to provision Certificates for HTTPRoutes.
// Default it to true so cert-manager works out of the box.
func TestDefaultConfigurationEnablesHTTPRouteGatewaySyncByDefault(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
			},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("DefaultConfiguration returned error: %v", err)
	}

	if !defaulted.Spec.FeatureGates[features.HTTPRouteGatewaySync] {
		t.Fatalf("expected HTTPRouteGatewaySync feature gate to default to true, got %v", defaulted.Spec.FeatureGates[features.HTTPRouteGatewaySync])
	}
}

// TestDefaultConfigurationPreservesExplicitHTTPRouteGatewaySyncFalse ensures an
// admin who explicitly disabled the feature gate keeps that choice — the
// defaulting only applies when the gate is unset.
func TestDefaultConfigurationPreservesExplicitHTTPRouteGatewaySyncFalse(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
			},
			FeatureGates: map[string]bool{
				features.HTTPRouteGatewaySync: false,
			},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("DefaultConfiguration returned error: %v", err)
	}

	if defaulted.Spec.FeatureGates[features.HTTPRouteGatewaySync] {
		t.Fatalf("expected explicit HTTPRouteGatewaySync=false to be preserved, got true")
	}
}

func TestDefaultConfigurationPreservesExplicitGatewayClassNameWhenExternalGatewayIsSet(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "kkp.example.com",
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ClassName: "my-custom-gatewayclass",
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	defaulted, err := defaulting.DefaultConfiguration(cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("DefaultConfiguration returned error: %v", err)
	}

	if defaulted.Spec.Ingress.Gateway.ClassName != "my-custom-gatewayclass" {
		t.Fatalf("expected explicit ClassName to be preserved so validation can reject the conflict, got %q", defaulted.Spec.Ingress.Gateway.ClassName)
	}
}
