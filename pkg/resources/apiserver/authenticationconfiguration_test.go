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

package apiserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeAuthConfigData struct {
	cluster   *kubermaticv1.Cluster
	issuerURL string
	clientID  string
	authYAML  string
}

func (f *fakeAuthConfigData) Cluster() *kubermaticv1.Cluster          { return f.cluster }
func (f *fakeAuthConfigData) OIDCIssuerURL() string                   { return f.issuerURL }
func (f *fakeAuthConfigData) OIDCIssuerClientID() string              { return f.clientID }
func (f *fakeAuthConfigData) AuthenticationConfigurationYAML() []byte { return []byte(f.authYAML) }

func TestAuthenticationConfigurationReconciler(t *testing.T) {
	tests := []struct {
		name                     string
		data                     *fakeAuthConfigData
		caBundle                 string
		enableOIDCAuthentication bool
		existingSecret           *corev1.Secret
		expectName               string
		expectError              string
		expectYAMLContains       []string
		expectDataRaw            string
	}{
		{
			name: "custom secret exists with correct key",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					AuthenticationConfiguration: &kubermaticv1.AuthenticationConfiguration{
						SecretName: "my-auth-config",
						SecretKey:  "config.yaml",
					},
				}},
			},
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: time.Now()}},
				Data:       map[string][]byte{"config.yaml": []byte("test")},
			},
			expectName: "my-auth-config",
		},
		{
			name: "custom secret does not exist",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
					AuthenticationConfiguration: &kubermaticv1.AuthenticationConfiguration{
						SecretName: "missing-secret",
						SecretKey:  "config.yaml",
					},
				}},
			},
			existingSecret: &corev1.Secret{},
			expectName:     "missing-secret",
			expectError:    "non-existing secret",
		},
		{
			name: "custom secret missing required key",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
					AuthenticationConfiguration: &kubermaticv1.AuthenticationConfiguration{
						SecretName: "my-auth-config",
						SecretKey:  "config.yaml",
					},
				}},
			},
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: time.Now()}},
				Data:       map[string][]byte{"wrong-key": []byte("test")},
			},
			expectName:  "my-auth-config",
			expectError: "does not specify the required key",
		},
		{
			name: "OIDC from cluster spec on k8s 1.34+",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
					OIDC: kubermaticv1.OIDCSettings{
						IssuerURL:      "https://issuer.example.com",
						ClientID:       "my-client",
						UsernameClaim:  "email",
						UsernamePrefix: "oidc:",
						GroupsClaim:    "groups",
						GroupsPrefix:   "oidc:",
					},
				}},
			},
			caBundle:   "ca-data",
			expectName: resources.ApiserverAuthenticationConfigurationSecretName,
			expectYAMLContains: []string{
				"kind: AuthenticationConfiguration",
				"apiVersion: apiserver.config.k8s.io/v1",
				"url: https://issuer.example.com",
				"audiences:\n    - my-client",
				"certificateAuthority: ca-data",
			},
		},
		{
			name: "OIDC from cluster spec on k8s <1.34",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.33.10"),
					OIDC: kubermaticv1.OIDCSettings{
						IssuerURL:      "https://issuer.example.com",
						ClientID:       "my-client",
						UsernameClaim:  "email",
						UsernamePrefix: "oidc:",
						GroupsClaim:    "groups",
						GroupsPrefix:   "oidc:",
					},
				}},
			},
			caBundle:   "ca-data",
			expectName: resources.ApiserverAuthenticationConfigurationSecretName,
			expectYAMLContains: []string{
				"kind: AuthenticationConfiguration",
				"apiVersion: apiserver.config.k8s.io/v1beta1",
				"url: https://issuer.example.com",
				"audiences:\n    - my-client",
				"certificateAuthority: ca-data",
			},
		},
		{
			name: "OIDC from cluster spec with required claims",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
					OIDC: kubermaticv1.OIDCSettings{
						IssuerURL:     "https://issuer.example.com",
						ClientID:      "my-client",
						RequiredClaim: "aud=my-aud,sub=my-sub",
					},
				}},
			},
			expectName: resources.ApiserverAuthenticationConfigurationSecretName,
			expectYAMLContains: []string{
				"claim: aud",
				"requiredValue: my-aud",
				"claim: sub",
				"requiredValue: my-sub",
			},
		},
		{
			name: "OIDC from cluster spec with malformed required claim",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
					OIDC: kubermaticv1.OIDCSettings{
						IssuerURL:     "https://issuer.example.com",
						ClientID:      "my-client",
						RequiredClaim: "malformed",
					},
				}},
			},
			expectName:  resources.ApiserverAuthenticationConfigurationSecretName,
			expectError: "malformed",
		},
		{
			name: "seed authentication configuration YAML passthrough",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
				}},
				authYAML: "kind: AuthenticationConfiguration\napiVersion: apiserver.config.k8s.io/v1\njwt: []\n",
			},
			expectName:    resources.ApiserverAuthenticationConfigurationSecretName,
			expectDataRaw: "kind: AuthenticationConfiguration\napiVersion: apiserver.config.k8s.io/v1\njwt: []\n",
		},
		{
			name: "enable OIDC authentication from seed",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
				}},
				issuerURL: "https://seed-issuer.example.com",
				clientID:  "seed-client",
			},
			caBundle:                 "seed-ca",
			enableOIDCAuthentication: true,
			expectName:               resources.ApiserverAuthenticationConfigurationSecretName,
			expectYAMLContains: []string{
				"kind: AuthenticationConfiguration",
				"apiVersion: apiserver.config.k8s.io/v1",
				"url: https://seed-issuer.example.com",
				"audiences:\n    - seed-client",
				"certificateAuthority: seed-ca",
				"claim: email",
				"claim: groups",
				"prefix: 'oidc:'",
			},
		},
		{
			name: "no OIDC configured produces empty JWT list",
			data: &fakeAuthConfigData{
				cluster: &kubermaticv1.Cluster{Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("1.34.0"),
				}},
			},
			expectName: resources.ApiserverAuthenticationConfigurationSecretName,
			expectYAMLContains: []string{
				"kind: AuthenticationConfiguration",
				"apiVersion: apiserver.config.k8s.io/v1",
				"jwt: []",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := AuthenticationConfigurationReconciler(tt.data, tt.caBundle, tt.enableOIDCAuthentication)
			name, reconciler := factory()
			require.Equal(t, tt.expectName, name)

			existing := tt.existingSecret
			if existing == nil {
				existing = &corev1.Secret{}
			}

			secret, err := reconciler(existing)
			if tt.expectError != "" {
				require.ErrorContains(t, err, tt.expectError)
				return
			} else {
				require.NoError(t, err)
			}

			if tt.expectDataRaw != "" {
				got := string(secret.Data[resources.AuthenticationConfigurationKey])
				require.Equal(t, tt.expectDataRaw, got, "AuthenticationConfiguration YAML")
			}

			yamlData := string(secret.Data[resources.AuthenticationConfigurationKey])
			for _, s := range tt.expectYAMLContains {
				require.Contains(t, yamlData, s+"\n", "AuthenticationConfigurationYAML should contain")
			}
		})
	}
}
