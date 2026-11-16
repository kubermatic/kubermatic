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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestLocalOIDCClientSecretsMatch makes sure that the secrets generated for the
// KubermaticConfiguration and the Dex static clients agree, for both OIDC
// clients, based on the shipped example files.
func TestLocalOIDCClientSecretsMatch(t *testing.T) {
	dir := t.TempDir()

	for src, dst := range map[string]string{
		"kubermatic.example.ce.yaml": "kubermatic.example.yaml",
		"values.example.ce.yaml":     "values.example.yaml",
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "charts", src))
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(dir, dst), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	secrets := newOIDCClientSecrets()

	kubermaticPath, err := prepareKubermaticConfiguration(dir, "127.0.0.1.nip.io", secrets)
	if err != nil {
		t.Fatalf("failed to prepare KubermaticConfiguration: %v", err)
	}

	valuesPath, err := prepareHelmValues(dir, "127.0.0.1.nip.io", secrets)
	if err != nil {
		t.Fatalf("failed to prepare Helm values: %v", err)
	}

	var config struct {
		Spec struct {
			Auth struct {
				ClientID           string `yaml:"clientID"`
				ClientSecret       string `yaml:"clientSecret"`
				IssuerClientID     string `yaml:"issuerClientID"`
				IssuerClientSecret string `yaml:"issuerClientSecret"`
			} `yaml:"auth"`
		} `yaml:"spec"`
	}
	readYAML(t, kubermaticPath, &config)

	var values struct {
		Dex struct {
			Config struct {
				StaticClients []struct {
					ID     string `yaml:"id"`
					Secret string `yaml:"secret"`
				} `yaml:"staticClients"`
			} `yaml:"config"`
		} `yaml:"dex"`
	}
	readYAML(t, valuesPath, &values)

	auth := config.Spec.Auth
	if auth.ClientSecret == "" || auth.IssuerClientSecret == "" {
		t.Fatalf("expected both client secrets to be set, got %+v", auth)
	}

	if auth.ClientSecret == auth.IssuerClientSecret {
		t.Error("the login and the issuer client must not share a secret")
	}

	dexSecrets := map[string]string{}
	for _, client := range values.Dex.Config.StaticClients {
		dexSecrets[client.ID] = client.Secret
	}

	for name, want := range map[string]string{
		auth.ClientID:       auth.ClientSecret,
		auth.IssuerClientID: auth.IssuerClientSecret,
	} {
		if got, ok := dexSecrets[name]; !ok {
			t.Errorf("Dex client %q is missing in the Helm values", name)
		} else if got != want {
			t.Errorf("Dex client %q has a different secret than the KubermaticConfiguration", name)
		}
	}
}

func readYAML(t *testing.T, path string, out any) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := yaml.Unmarshal(data, out); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
}
