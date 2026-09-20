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
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLimaVMTemplateDefaults(t *testing.T) {
	template := limaVMTemplate("kkp-cluster", 8, 16, 60, nil)

	for _, needle := range []string{
		"cpus: 8",
		`memory: "16GiB"`,
		`disk: "60GiB"`,
		"# instance: kkp-kkp-cluster",
	} {
		if !strings.Contains(template, needle) {
			t.Errorf("template does not contain %q:\n%s", needle, template)
		}
	}

	for _, needle := range []string{
		"- guestPort: 5000\n  hostPort: 5000",
		"- guestPort: 36443\n  hostPort: 36443",
		"- guestPort: 80\n  hostPort: 80",
		"- guestPort: 443\n  hostPort: 443",
		"- guestPort: 6443\n  hostPort: 6443",
		"- guestPort: 8088\n  hostPort: 8088",
	} {
		if !strings.Contains(template, needle) {
			t.Errorf("default template does not forward %q:\n%s", needle, template)
		}
	}

	if err := yaml.Unmarshal([]byte(template), &map[string]interface{}{}); err != nil {
		t.Errorf("template is not valid YAML: %v", err)
	}
}

func TestLimaVMTemplateOverrides(t *testing.T) {
	template := limaVMTemplate("my-kkp", 2, 4, 10, map[string]int{"http": 18080, "apiserver": 16443})

	for _, needle := range []string{
		"cpus: 2",
		`memory: "4GiB"`,
		`disk: "10GiB"`,
		"# instance: kkp-my-kkp",
		"- guestPort: 18080\n  hostPort: 18080",
		"- guestPort: 16443\n  hostPort: 16443",
	} {
		if !strings.Contains(template, needle) {
			t.Errorf("template does not contain %q:\n%s", needle, template)
		}
	}

	if strings.Contains(template, "guestPort: 80\n") {
		t.Errorf("overridden http port 80 should have been replaced:\n%s", template)
	}

	if err := yaml.Unmarshal([]byte(template), &map[string]interface{}{}); err != nil {
		t.Errorf("template is not valid YAML: %v", err)
	}
}
