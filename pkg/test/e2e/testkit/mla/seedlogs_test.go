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

package mla

import (
	"maps"
	"testing"
)

func TestParseLokiLabelDiscards(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  string
		expected map[string]float64
	}{
		{
			name:     "no discards",
			metrics:  "# HELP loki_build_info x\nloki_build_info{version=\"3.6.7\"} 1\n",
			expected: map[string]float64{},
		},
		{
			// copied from seed Loki 3.6.7 with max_label_names_per_series at 15
			name: "label limit across tenants",
			metrics: `# TYPE loki_discarded_samples_total counter
loki_discarded_samples_total{format="loki",policy="",reason="max_label_names_per_series",retention_hours="744",tenant="fake"} 6504
loki_discarded_samples_total{format="loki",policy="",reason="max_label_names_per_series",retention_hours="744",tenant="other"} 6
loki_discarded_samples_total{format="loki",policy="",reason="invalid_labels",retention_hours="744",tenant="fake"} 6504
`,
			expected: map[string]float64{"max_label_names_per_series": 6510, "invalid_labels": 6504},
		},
		{
			name:     "rate limits are ignored",
			metrics:  `loki_discarded_samples_total{reason="rate_limited",tenant="fake"} 12` + "\n",
			expected: map[string]float64{},
		},
		{
			name:     "trailing timestamp",
			metrics:  `loki_discarded_samples_total{reason="label_value_too_long",tenant="fake"} 3 1761580800000` + "\n",
			expected: map[string]float64{"label_value_too_long": 3},
		},
		{
			name:     "other label ending in reason",
			metrics:  `loki_discarded_samples_total{policy_reason="x",reason="duplicate_label_names"} 1` + "\n",
			expected: map[string]float64{"duplicate_label_names": 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			discards, err := parseLokiLabelDiscards(tc.metrics)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !maps.Equal(discards, tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, discards)
			}
		})
	}
}
