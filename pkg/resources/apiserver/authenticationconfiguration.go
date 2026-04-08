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
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiserverv1 "k8s.io/apiserver/pkg/apis/apiserver/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type authenticationConfigurationReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	OIDCIssuerURL() string
	OIDCIssuerClientID() string
	AuthenticationConfigurationYAML() []byte
}

func AuthenticationConfigurationReconciler(data authenticationConfigurationReconcilerData, caBundle string, enableOIDCAuthentication bool) reconciling.NamedSecretReconcilerFactory {
	cluster := data.Cluster()

	return func() (string, reconciling.SecretReconciler) {
		if cluster.Spec.IsAuthenticationConfigurationEnabled() {
			// User defined the Secret explicitly, don't modify it but ensure it exists.
			return cluster.Spec.AuthenticationConfiguration.SecretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
				if existing.CreationTimestamp.IsZero() {
					return nil, fmt.Errorf("spec.authenticationConfiguration.secretName points to non-existing secret %s", cluster.Spec.AuthenticationConfiguration.SecretName)
				}
				if _, ok := existing.Data[cluster.Spec.AuthenticationConfiguration.SecretKey]; !ok {
					return nil, fmt.Errorf("secret %s does not specify the required key %q", cluster.Spec.AuthenticationConfiguration.SecretName, cluster.Spec.AuthenticationConfiguration.SecretKey)
				}
				return existing, nil
			}
		}

		// Generate the Secret from the Cluster spec or the seed configuration as a fallback.
		return resources.ApiserverAuthenticationConfigurationSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			cfg := apiserverv1.AuthenticationConfiguration{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AuthenticationConfiguration",
					APIVersion: "apiserver.config.k8s.io/v1",
				},
				JWT: []apiserverv1.JWTAuthenticator{},
			}
			oidcSettings := cluster.Spec.OIDC //nolint:staticcheck
			seedAuthConf := data.AuthenticationConfigurationYAML()

			if data.Cluster().Spec.Version.LessThan(semver.NewSemverOrDie("1.34.0")) {
				// Kubernetes 1.30-1.33 only supports the v1beta1 version of the AuthenticationConfiguration API
				cfg.APIVersion = "apiserver.config.k8s.io/v1beta1"
			}

			switch {
			case oidcSettings.IssuerURL != "" && oidcSettings.ClientID != "":
				// Old way of integrating OIDC: based on Cluster.spec.OIDC.
				cfg.JWT = []apiserverv1.JWTAuthenticator{{
					Issuer: apiserverv1.Issuer{
						URL:                  oidcSettings.IssuerURL,
						Audiences:            []string{oidcSettings.ClientID},
						CertificateAuthority: caBundle,
					},
					ClaimMappings: apiserverv1.ClaimMappings{
						Username: apiserverv1.PrefixedClaimOrExpression{
							Claim:  oidcSettings.UsernameClaim,
							Prefix: ptr.To(oidcSettings.UsernamePrefix),
						},
						Groups: apiserverv1.PrefixedClaimOrExpression{
							Claim:  oidcSettings.GroupsClaim,
							Prefix: ptr.To(oidcSettings.GroupsPrefix),
						},
					},
				}}

				claimRules := strings.Split(oidcSettings.RequiredClaim, ",")
				validationRules := make([]apiserverv1.ClaimValidationRule, 0, len(claimRules))

				for _, rule := range claimRules {
					if rule = strings.TrimSpace(rule); rule != "" {
						kv := strings.SplitN(rule, "=", 2)
						if len(kv) != 2 {
							return nil, fmt.Errorf("spec.oidc.requiredClaim %q is malformed, expecting comma-separated key=value pairs", oidcSettings.RequiredClaim)
						}

						validationRules = append(validationRules, apiserverv1.ClaimValidationRule{
							Claim:         kv[0],
							RequiredValue: kv[1],
						})
					}
				}

				if len(validationRules) > 0 {
					cfg.JWT[0].ClaimValidationRules = validationRules
				}
			case len(seedAuthConf) > 0:
				secret.Data = map[string][]byte{resources.AuthenticationConfigurationKey: seedAuthConf}

				return secret, nil
			case enableOIDCAuthentication:
				usernameClaimPrefix := ""
				groupClaimPrefix := "oidc:"
				cfg.JWT = []apiserverv1.JWTAuthenticator{{
					Issuer: apiserverv1.Issuer{
						URL:                  data.OIDCIssuerURL(),
						Audiences:            []string{data.OIDCIssuerClientID()},
						CertificateAuthority: caBundle,
					},
					ClaimMappings: apiserverv1.ClaimMappings{
						Username: apiserverv1.PrefixedClaimOrExpression{
							Claim:  "email",
							Prefix: &usernameClaimPrefix,
						},
						Groups: apiserverv1.PrefixedClaimOrExpression{
							Claim:  "groups",
							Prefix: &groupClaimPrefix,
						},
					},
				}}
			}

			b, err := yaml.Marshal(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal AuthenticationConfiguration: %w", err)
			}

			secret.Data = map[string][]byte{resources.AuthenticationConfigurationKey: b}

			return secret, nil
		}
	}
}
