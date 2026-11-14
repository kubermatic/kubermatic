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

package validation

import (
	"context"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateLoginClient(t *testing.T) {
	seedWithClient := func(name, clientID string) *kubermaticv1.Seed {
		return &kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: kubermaticv1.SeedSpec{
				OIDCProviderConfiguration: &kubermaticv1.OIDCProviderConfiguration{
					IssuerURL:      "https://issuer.example.com",
					IssuerClientID: clientID,
				},
			},
		}
	}

	config := func(clientID, issuerClientID string) *kubermaticv1.KubermaticConfiguration {
		return &kubermaticv1.KubermaticConfiguration{
			Spec: kubermaticv1.KubermaticConfigurationSpec{
				Auth: kubermaticv1.KubermaticAuthConfiguration{
					ClientID:       clientID,
					ClientSecret:   "secret",
					IssuerClientID: issuerClientID,
				},
			},
		}
	}

	headless := config("kubermatic", "kubermaticIssuer")
	headless.Spec.FeatureGates = map[string]bool{features.HeadlessInstallation: true}

	testCases := []struct {
		name      string
		seeds     []*kubermaticv1.Seed
		oldConfig *kubermaticv1.KubermaticConfiguration
		config    *kubermaticv1.KubermaticConfiguration
		wantErr   string
	}{
		{
			name:   "no Seeds",
			config: config("kubermatic", "kubermaticIssuer"),
		},
		{
			name:   "Seeds with other clients",
			seeds:  []*kubermaticv1.Seed{seedWithClient("europe", "europeIssuer"), {ObjectMeta: metav1.ObjectMeta{Name: "us"}}},
			config: config("kubermatic", "kubermaticIssuer"),
		},
		{
			name:    "Seed using the login client",
			seeds:   []*kubermaticv1.Seed{seedWithClient("europe", "kubermatic"), seedWithClient("asia", "kubermatic")},
			config:  config("kubermatic", "kubermaticIssuer"),
			wantErr: "Seed(s) asia, europe",
		},
		{
			name:    "Seed using the defaulted login client",
			seeds:   []*kubermaticv1.Seed{seedWithClient("europe", "kubermatic")},
			config:  &kubermaticv1.KubermaticConfiguration{Spec: kubermaticv1.KubermaticConfigurationSpec{Auth: kubermaticv1.KubermaticAuthConfiguration{ClientSecret: "secret"}}},
			wantErr: "Seed(s) europe",
		},
		{
			name:   "single-client mode",
			seeds:  []*kubermaticv1.Seed{seedWithClient("europe", "kubermaticIssuer")},
			config: config("kubermaticIssuer", "kubermaticIssuer"),
		},
		{
			name:   "headless installation",
			seeds:  []*kubermaticv1.Seed{seedWithClient("europe", "kubermatic")},
			config: headless,
		},
		{
			name:      "update switching from single-client mode",
			seeds:     []*kubermaticv1.Seed{seedWithClient("europe", "kubermaticIssuer")},
			oldConfig: config("kubermaticIssuer", "kubermaticIssuer"),
			config:    config("kubermaticIssuer", "otherIssuer"),
			wantErr:   "Seed(s) europe",
		},
		{
			name:      "update changing the login client",
			seeds:     []*kubermaticv1.Seed{seedWithClient("europe", "europeIssuer")},
			oldConfig: config("kubermatic", "kubermaticIssuer"),
			config:    config("europeIssuer", "kubermaticIssuer"),
			wantErr:   "Seed(s) europe",
		},
		{
			name:      "unrelated update with a Seed that already uses the login client",
			seeds:     []*kubermaticv1.Seed{seedWithClient("europe", "kubermatic")},
			oldConfig: config("kubermatic", "kubermaticIssuer"),
			config:    config("kubermatic", "kubermaticIssuer"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator(test.NewSeedsGetter(tc.seeds...))

			var err error
			if tc.oldConfig == nil {
				_, err = v.ValidateCreate(context.Background(), tc.config)
			} else {
				_, err = v.ValidateUpdate(context.Background(), tc.oldConfig, tc.config)
			}

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), "spec.auth.clientID") || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Expected an error on spec.auth.clientID naming %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
