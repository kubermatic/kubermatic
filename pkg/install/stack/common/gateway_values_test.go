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

package common

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultMasterHTTPRouteGatewayValues(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	testCases := []struct {
		name          string
		values        string
		wantChanged   bool
		wantName      string
		wantNamespace string
	}{
		{
			name: "defaults built-in Gateway values",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantChanged:   true,
			wantName:      "platform-gateway",
			wantNamespace: "networking",
		},
		{
			name: "preserves explicit Gateway values",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: seed-gateway
  gatewayNamespace: seed-ingress
`,
			wantChanged:   false,
			wantName:      "seed-gateway",
			wantNamespace: "seed-ingress",
		},
		{
			name: "preserves custom Gateway name in default namespace",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: seed-gateway
  gatewayNamespace: kubermatic
`,
			wantChanged:   false,
			wantName:      "seed-gateway",
			wantNamespace: "kubermatic",
		},
		{
			name: "preserves default Gateway name in custom namespace",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: seed-ingress
`,
			wantChanged:   false,
			wantName:      "kubermatic",
			wantNamespace: "seed-ingress",
		},
		{
			name: "skips when Gateway API migration is disabled",
			values: `
migrateGatewayAPI: false
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantChanged:   false,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := yamled.Load(strings.NewReader(tc.values))
			if err != nil {
				t.Fatalf("failed to load Helm values: %v", err)
			}

			if gotChanged := DefaultMasterHTTPRouteGatewayValues(cfg, doc, logrus.New()); gotChanged != tc.wantChanged {
				t.Fatalf("DefaultMasterHTTPRouteGatewayValues() = %v, want %v", gotChanged, tc.wantChanged)
			}

			gatewayName, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayName"})
			if gatewayName != tc.wantName {
				t.Errorf("expected Gateway name %s, got %s", tc.wantName, gatewayName)
			}

			gatewayNamespace, _ := doc.GetString(yamled.Path{"httpRoute", "gatewayNamespace"})
			if gatewayNamespace != tc.wantNamespace {
				t.Errorf("expected Gateway namespace %s, got %s", tc.wantNamespace, gatewayNamespace)
			}
		})
	}
}

func TestMasterHTTPRouteGatewayReference(t *testing.T) {
	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
					ExternalGateway: &kubermaticv1.KubermaticExternalGatewayReference{
						Name:      "platform-gateway",
						Namespace: "networking",
					},
				},
			},
		},
	}

	testCases := []struct {
		name          string
		values        string
		wantName      string
		wantNamespace string
	}{
		{
			name: "default values resolve to external Gateway",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantName:      "platform-gateway",
			wantNamespace: "networking",
		},
		{
			name: "explicit values are preserved",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: dex-gateway
  gatewayNamespace: dex-networking
`,
			wantName:      "dex-gateway",
			wantNamespace: "dex-networking",
		},
		{
			name: "migration disabled uses configured Helm values",
			values: `
migrateGatewayAPI: false
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := yamled.Load(strings.NewReader(tc.values))
			if err != nil {
				t.Fatalf("failed to load Helm values: %v", err)
			}

			got := MasterHTTPRouteGatewayReference(cfg, doc)
			if got.Name != tc.wantName {
				t.Fatalf("expected Gateway name %s, got %s", tc.wantName, got.Name)
			}
			if got.Namespace != tc.wantNamespace {
				t.Fatalf("expected Gateway namespace %s, got %s", tc.wantNamespace, got.Namespace)
			}
		})
	}
}
