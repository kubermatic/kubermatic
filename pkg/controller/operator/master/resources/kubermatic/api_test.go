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

package kubermatic

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
)

func TestAPIDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	tests := []schedulingTestCase{
		{
			name:  "all scheduling fields set",
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name: "no scheduling fields",
			want: nil,
		},
		{
			name: "only nodeSelector",
			input: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector: map[string]string{"role": "test"},
			},
			want: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector: map[string]string{"role": "test"},
			},
		},
		{
			name: "overwrites pre-existing scheduling fields",
			preExisting: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector:      map[string]string{"old": "value"},
				PriorityClassName: "old-priority",
			},
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name:        "clears pre-existing fields when config is empty",
			preExisting: ptrScheduling(fullScheduling()),
			want:        nil,
		},
	}

	setSched := func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
		cfg.Spec.API.PodSchedulingConfigurations = s
	}
	newCreator := func(cfg *kubermaticv1.KubermaticConfiguration) func(*appsv1.Deployment) (*appsv1.Deployment, error) {
		_, c := APIDeploymentReconciler(cfg, "", versions)()
		return c
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSchedulingTest(t, tt, setSched, newCreator)
		})
	}
}

func TestAPIDeploymentOIDCAuthenticatorClientSecret(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	tests := []struct {
		name string
		auth kubermaticv1.KubermaticAuthConfiguration
		want string
	}{
		{
			name: "same client, no clientSecret: uses issuerClientSecret",
			auth: kubermaticv1.KubermaticAuthConfiguration{
				ClientID:           "kubermaticIssuer",
				IssuerClientID:     "kubermaticIssuer",
				IssuerClientSecret: "issuer-secret",
			},
			want: "issuer-secret",
		},
		{
			name: "same client: clientSecret is ignored, one client has one secret",
			auth: kubermaticv1.KubermaticAuthConfiguration{
				ClientID:           "kubermaticIssuer",
				ClientSecret:       "<placeholder>",
				IssuerClientID:     "kubermaticIssuer",
				IssuerClientSecret: "issuer-secret",
			},
			want: "issuer-secret",
		},
		{
			name: "separate clients: uses clientSecret",
			auth: kubermaticv1.KubermaticAuthConfiguration{
				ClientID:           "kubermatic",
				ClientSecret:       "login-secret",
				IssuerClientID:     "kubermaticIssuer",
				IssuerClientSecret: "issuer-secret",
			},
			want: "login-secret",
		},
		{
			name: "separate clients, no clientSecret: never reuses issuerClientSecret",
			auth: kubermaticv1.KubermaticAuthConfiguration{
				ClientID:           "kubermatic",
				IssuerClientID:     "kubermaticIssuer",
				IssuerClientSecret: "issuer-secret",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &kubermaticv1.KubermaticConfiguration{}
			cfg.Spec.Auth = tt.auth
			cfg.Spec.API.PProfEndpoint = ptr.To(":6600")

			_, creator := APIDeploymentReconciler(cfg, "", versions)()
			dep, err := creator(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("failed to reconcile deployment: %v", err)
			}

			want := "-oidc-authenticator-client-secret=" + tt.want

			for _, arg := range dep.Spec.Template.Spec.Containers[0].Args {
				if arg == want {
					return
				}
			}

			t.Errorf("expected argument %q, got %v", want, dep.Spec.Template.Spec.Containers[0].Args)
		})
	}
}
