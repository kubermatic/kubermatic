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
	"bytes"
	"strings"
	"testing"
)

const statefulSet = `---
# Source: cortex/templates/ingester.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cortex-ingester
  namespace: mla
spec:
  replicas: {{REPLICAS}}
  serviceName: cortex-ingester-headless
  selector:
    matchLabels:
      app: {{APP}}
  volumeClaimTemplates:
    - metadata:
        name: storage
      spec:
        accessModes: [ReadWriteOnce]
        resources:
          requests:
            storage: {{STORAGE}}
`

const service = `---
apiVersion: v1
kind: Service
metadata:
  name: cortex-ingester-headless
  namespace: mla
spec:
  clusterIP: None
`

const hookJob = `---
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
  annotations:
    helm.sh/hook: pre-upgrade
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: {{IMAGE}}
`

func render(tpl string, replacements ...string) string {
	return strings.NewReplacer(replacements...).Replace(tpl)
}

func sts(replicas, app, storage string) string {
	return render(statefulSet, "{{REPLICAS}}", replicas, "{{APP}}", app, "{{STORAGE}}", storage)
}

func TestCheck(t *testing.T) {
	baseline := sts("1", "ingester", "10Gi") + service

	testcases := []struct {
		name         string
		base         *string
		current      *string
		allowlist    string
		wantOK       bool
		wantErr      bool
		wantInOutput []string
	}{
		{
			name:         "selector change fails",
			base:         &baseline,
			current:      new(sts("1", "cortex-ingester", "10Gi") + service),
			wantOK:       false,
			wantInOutput: []string{"FAIL test.yaml.out: StatefulSet/mla/cortex-ingester spec.selector", `"app":"ingester"`, `"app":"cortex-ingester"`},
		},
		{
			name:    "replicas change passes",
			base:    &baseline,
			current: new(sts("3", "ingester", "10Gi") + service),
			wantOK:  true,
		},
		{
			name:    "storage written in bytes is equal",
			base:    &baseline,
			current: new(sts("1", "ingester", "10737418240") + service),
			wantOK:  true,
		},
		{
			name:    "hook job is ignored",
			base:    new(baseline + render(hookJob, "{{IMAGE}}", "a:1")),
			current: new(baseline + render(hookJob, "{{IMAGE}}", "a:2")),
			wantOK:  true,
		},
		{
			name:         "non-hook job template change fails",
			base:         new(baseline + strings.Replace(render(hookJob, "{{IMAGE}}", "a:1"), "    helm.sh/hook: pre-upgrade\n", "    foo: bar\n", 1)),
			current:      new(baseline + strings.Replace(render(hookJob, "{{IMAGE}}", "a:2"), "    helm.sh/hook: pre-upgrade\n", "    foo: bar\n", 1)),
			wantOK:       false,
			wantInOutput: []string{"Job/migrate spec.template"},
		},
		{
			name:         "statefulset removal fails",
			base:         &baseline,
			current:      new(service),
			wantOK:       false,
			wantInOutput: []string{"StatefulSet/mla/cortex-ingester removed"},
		},
		{
			name:         "deleted file fails for statefulset",
			base:         &baseline,
			current:      nil,
			wantOK:       false,
			wantInOutput: []string{"StatefulSet/mla/cortex-ingester removed"},
		},
		{
			name:         "volume claim template storage change fails",
			base:         &baseline,
			current:      new(sts("1", "ingester", "20Gi") + service),
			wantOK:       false,
			wantInOutput: []string{"StatefulSet/mla/cortex-ingester spec.volumeClaimTemplates[storage]", `"storage":"10Gi"`, `"storage":"20Gi"`},
		},
		{
			name:         "service clusterIP change fails",
			base:         &baseline,
			current:      new(sts("1", "ingester", "10Gi") + strings.Replace(service, "None", "10.0.0.1", 1)),
			wantOK:       false,
			wantInOutput: []string{"Service/mla/cortex-ingester-headless spec.clusterIP"},
		},
		{
			name:    "allowlisted change passes",
			base:    &baseline,
			current: new(sts("1", "cortex-ingester", "10Gi") + service),
			allowlist: `
- object: StatefulSet/mla/cortex-ingester
  field: spec.selector
  reason: selector renamed
  migration: pkg/install/stack/usercluster-mla/stack.go:123
`,
			wantOK:       true,
			wantInOutput: []string{"ALLOWED test.yaml.out: StatefulSet/mla/cortex-ingester spec.selector"},
		},
		{
			name:    "allowlist entry without migration fails",
			base:    &baseline,
			current: new(sts("1", "cortex-ingester", "10Gi") + service),
			allowlist: `
- object: StatefulSet/mla/cortex-ingester
  field: spec.selector
  reason: selector renamed
`,
			wantErr: true,
		},
		{
			name:         "new file is skipped",
			base:         nil,
			current:      new(sts("1", "cortex-ingester", "10Gi")),
			wantOK:       true,
			wantInOutput: []string{"SKIP test.yaml.out: new file"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			allowlist, err := parseAllowlist([]byte(tc.allowlist))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected allowlist error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected allowlist error: %v", err)
			}

			pair := renderPair{Path: "test.yaml.out"}
			if tc.base != nil {
				pair.Base = []byte(*tc.base)
			}
			if tc.current != nil {
				pair.Current = []byte(*tc.current)
			}

			var out bytes.Buffer
			ok, err := check([]renderPair{pair}, allowlist, &out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ok != tc.wantOK {
				t.Errorf("expected ok=%v, got %v, output:\n%s", tc.wantOK, ok, out.String())
			}

			for _, want := range tc.wantInOutput {
				if !strings.Contains(out.String(), want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, out.String())
				}
			}
		})
	}
}
