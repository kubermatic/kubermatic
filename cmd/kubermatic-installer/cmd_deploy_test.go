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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestKubeconfig writes a kubeconfig with one cluster and context per given server.
func writeTestKubeconfig(t *testing.T, dir string, name string, servers map[string]string, currentContext string) string {
	t.Helper()

	var clusters, contexts strings.Builder
	for ctxName, server := range servers {
		fmt.Fprintf(&clusters, "- name: %s\n  cluster:\n    server: %s\n", ctxName, server)
		fmt.Fprintf(&contexts, "- name: %s\n  context:\n    cluster: %s\n    user: admin\n", ctxName, ctxName)
	}

	content := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
%scontexts:
%scurrent-context: %s
users:
- name: admin
  user:
    token: test
`, clusters.String(), contexts.String(), currentContext)

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	return path
}

func TestLoadKubeconfig(t *testing.T) {
	dir := t.TempDir()

	given := writeTestKubeconfig(t, dir, "given", map[string]string{
		"main":  "https://given.example.com",
		"other": "https://given-other.example.com",
	}, "main")
	extra := writeTestKubeconfig(t, dir, "extra", map[string]string{
		"extra": "https://extra.example.com",
	}, "extra")
	fromEnv := writeTestKubeconfig(t, dir, "from-env", map[string]string{
		"env": "https://from-env.example.com",
	}, "env")

	// $HOME/.kube/config and $KUBECONFIG must not be used when a kubeconfig is given.
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o755); err != nil {
		t.Fatalf("failed to create .kube directory: %v", err)
	}
	writeTestKubeconfig(t, filepath.Join(home, ".kube"), "config", map[string]string{
		"home": "https://home.example.com",
	}, "home")
	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", fromEnv)

	testCases := []struct {
		name           string
		kubeconfig     string
		kubeContext    string
		expectedServer string
		expectedErr    string
	}{
		{
			name:           "given kubeconfig wins over $KUBECONFIG and $HOME",
			kubeconfig:     given,
			expectedServer: "https://given.example.com",
		},
		{
			name:           "kube context selects a context from the given kubeconfig",
			kubeconfig:     given,
			kubeContext:    "other",
			expectedServer: "https://given-other.example.com",
		},
		{
			name:           "list of files is merged like $KUBECONFIG",
			kubeconfig:     given + string(os.PathListSeparator) + extra,
			kubeContext:    "extra",
			expectedServer: "https://extra.example.com",
		},
		{
			name:        "unknown kube context",
			kubeconfig:  given,
			kubeContext: "missing",
			expectedErr: `context "missing" does not exist`,
		},
		{
			name:        "missing file",
			kubeconfig:  filepath.Join(dir, "does-not-exist"),
			expectedErr: "no such file or directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := loadKubeconfig(tc.kubeconfig, tc.kubeContext)
			if tc.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got none", tc.expectedErr)
				}
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("expected error containing %q, got %v", tc.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if config.Host != tc.expectedServer {
				t.Errorf("expected server %q, got %q", tc.expectedServer, config.Host)
			}

			if config.QPS != -1 {
				t.Errorf("expected client-side rate limiting to be disabled (QPS -1), got %v", config.QPS)
			}
		})
	}
}
