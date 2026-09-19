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

func TestKindConfigContentDefaults(t *testing.T) {
	expected := `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
 - role: control-plane
 - role: worker
   extraPortMappings:
   # nodeport-proxy for user-cluster control-planes
   - containerPort: 30652
     hostPort: 6443
   - containerPort: 32121
     hostPort: 8088
   # envoy gateway for kubermatic api and dashboard
   - containerPort: 31514
     hostPort: 80
   - containerPort: 32394
     hostPort: 443
`

	if config := kindConfigContent(nil, ""); config != expected {
		t.Errorf("default kind config does not match the previous constant:\n%s", config)
	}

	if err := yaml.Unmarshal([]byte(expected), &map[string]interface{}{}); err != nil {
		t.Errorf("default kind config is not valid YAML: %v", err)
	}
}

func TestKindConfigContentOverrides(t *testing.T) {
	config := kindConfigContent(map[string]int{"http": 8080, "apiserver": 16443}, "/tmp/certs")

	for _, needle := range []string{"hostPort: 16443", "hostPort: 8088", "hostPort: 8080", "hostPort: 443"} {
		if !strings.Contains(config, needle) {
			t.Errorf("config does not contain %q:\n%s", needle, config)
		}
	}

	if count := strings.Count(config, "hostPath: /tmp/certs"); count != 2 {
		t.Errorf("expected registry mounts on both nodes, found %d:\n%s", count, config)
	}

	if err := yaml.Unmarshal([]byte(config), &map[string]interface{}{}); err != nil {
		t.Errorf("overridden kind config is not valid YAML: %v", err)
	}
}

func TestSplitImageOverride(t *testing.T) {
	testcases := []struct {
		value         string
		repository    string
		tag           string
		hasRepository bool
	}{
		{value: "dev1", repository: "", tag: "dev1", hasRepository: false},
		{value: "quay.io/kubermatic/kubermatic:dev1", repository: "quay.io/kubermatic/kubermatic", tag: "dev1", hasRepository: true},
		{value: "localhost:5000/kubermatic:dev1", repository: "localhost:5000/kubermatic", tag: "dev1", hasRepository: true},
		{value: "localhost:5000/kubermatic", repository: "localhost:5000/kubermatic", tag: "", hasRepository: true},
	}

	for _, testcase := range testcases {
		repository, tag, hasRepository := splitImageOverride(testcase.value)
		if repository != testcase.repository || tag != testcase.tag || hasRepository != testcase.hasRepository {
			t.Errorf("%q => (%q, %q, %v), expected (%q, %q, %v)",
				testcase.value, repository, tag, hasRepository, testcase.repository, testcase.tag, testcase.hasRepository)
		}
	}
}

func TestHostPortFallback(t *testing.T) {
	opt := LocalOptions{}
	if port := opt.hostPort("http"); port != 80 {
		t.Errorf("expected default http port 80, got %d", port)
	}

	opt.HostPorts = map[string]int{"http": 8080}
	if port := opt.hostPort("http"); port != 8080 {
		t.Errorf("expected http port 8080, got %d", port)
	}
}
