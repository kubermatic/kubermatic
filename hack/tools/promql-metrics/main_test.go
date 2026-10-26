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
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

var testdataArgs = []string{
	"-root", "testdata",
	"-rules", "rules/*.yaml",
	"-dashboards", "dashboards/**/*.json",
}

func runTool(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := run(append(slices.Clone(testdataArgs), args...), &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func TestTextOutput(t *testing.T) {
	code, stdout, stderr := runTool(t)
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}

	want := []string{
		"container_memory_working_set_bytes",
		"http_requests_total",
		"job:http_requests:rate5m",
		"kube_pod_info",
		"legacy_row_metric",
		"node_cpu_seconds_total",
		"node_network_receive_bytes_total",
		"node_uname_info",
		"process_cpu_seconds_total",
		"up",
	}

	got := strings.Fields(stdout)
	if !slices.Equal(got, want) {
		t.Errorf("metric names mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestJSONOutput(t *testing.T) {
	code, stdout, stderr := runTool(t, "-format", "json")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}

	var r report
	if err := json.Unmarshal([]byte(stdout), &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	for _, m := range r.Metrics {
		if m.Name == "up" && !slices.Equal(m.Files, []string{"rules/sample.yaml"}) {
			t.Errorf("unexpected files for up: %v", m.Files)
		}
	}

	wantSkipped := map[string]string{
		`{namespace="$namespace"} |= "error"`:      "loki datasource",
		`count_over_time({app="x"}[5m])`:           "loki datasource",
		`label_values({job="x"}, app)`:             "loki datasource",
		`label_values(namespace)`:                  "label_values without a metric selector",
		`sum(rate(broken_metric_total[5m])`:        "parse error",
		`{__name__=~"apiserver_.+_total"}`:         "non-equality __name__ matcher",
		`kube_pod_info{namespace="[[namespace]]"}`: "",
	}

	got := map[string]string{}
	for _, s := range r.Skipped {
		if s.File != "dashboards/sample.json" {
			t.Errorf("unexpected file %q in skipped entry", s.File)
		}
		got[s.Expr] = s.Reason
	}

	for expr, reason := range wantSkipped {
		gotReason, ok := got[expr]
		switch {
		case reason == "" && ok:
			t.Errorf("expression %q should not be skipped, got reason %q", expr, gotReason)
		case reason != "" && !ok:
			t.Errorf("expression %q should be skipped with %q", expr, reason)
		case reason != "" && !strings.HasPrefix(gotReason, reason):
			t.Errorf("expression %q skipped with %q, want prefix %q", expr, gotReason, reason)
		}
	}

	if len(r.Skipped) != 6 {
		t.Errorf("got %d skipped expressions, want 6: %+v", len(r.Skipped), r.Skipped)
	}
}

func TestContractCheckPass(t *testing.T) {
	code, stdout, stderr := runTool(t, "-check", "-contract", "testdata/contract-pass.yaml")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}

	if !strings.Contains(stdout, "metric contract OK") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

func TestContractCheckFail(t *testing.T) {
	code, _, stderr := runTool(t, "-check", "-contract", "testdata/contract-fail.yaml")
	if code != 1 {
		t.Fatalf("exit code %d, want 1, stderr: %s", code, stderr)
	}

	for _, want := range []string{
		"missing: container_memory_working_set_bytes is referenced by dashboards/sample.json",
		"missing: process_cpu_seconds_total",
		"stale: removed_metric_total",
		"duplicate: up is listed more than once (seed, userCluster)",
		`notInCI entry "legacy_row_metric" has no reason`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not contain %q:\n%s", want, stderr)
		}
	}
}

func TestContractCheckRequiresContract(t *testing.T) {
	if code, _, _ := runTool(t, "-check"); code != 2 {
		t.Errorf("exit code %d, want 2", code)
	}
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`rate(x[$__rate_interval])`, `rate(x[5m])`},
		{`rate(x[${__interval}])`, `rate(x[5m])`},
		{`increase(x[$__range])`, `increase(x[5m])`},
		{`rate(x[$interval])`, `rate(x[5m])`},
		{`x{pod=~"$pod", ns="${namespace:regex}"}`, `x{pod=~".+", ns=".+"}`},
		{`x{pod=~"$deployment-.*"}`, `x{pod=~".+-.*"}`},
		{`x{ns="[[namespace]]"}`, `x{ns=".+"}`},
		{`x{ns=$namespace}`, `x{ns=".+"}`},
		{`x{ns=~ [[namespace]]}`, `x{ns=~ ".+"}`},
		{`x > ${min_duration}`, `x > 1`},
		{`x{re="^a$"}`, `x{re="^a$"}`},
		{`x{a="\"$v\""}`, `x{a="\".+\""}`},
		{`x / $__range_s`, `x / 300`},
	}

	for _, tt := range tests {
		if got := substituteVariables(tt.in); got != tt.want {
			t.Errorf("substituteVariables(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGlobToRegexp(t *testing.T) {
	re, err := globToRegexp("charts/**/*.json")
	if err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]bool{
		"charts/a.json":     true,
		"charts/x/y/a.json": true,
		"charts/a.yaml":     false,
		"other/a.json":      false,
	} {
		if got := re.MatchString(path); got != want {
			t.Errorf("match %q = %v, want %v", path, got, want)
		}
	}
}
