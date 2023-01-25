/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package collectors

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

func convertToPrometheusLabels(labelKeys sets.Set[string]) []string {
	promLabels := sets.New[string]()
	for _, key := range sets.List(labelKeys) {
		// due to conversion, different labels might result in the same Prometheus label
		// (e.g. "foo-bar" and "foo/bar" will both be normalised to "foo_bar"), hence we
		// use a set.
		promLabels.Insert(convertToPrometheusLabel(key))
	}

	return sets.List(promLabels)
}

var validMetricLabel = regexp.MustCompile(`[^a-z0-9_]`)

func convertToPrometheusLabel(label string) string {
	return "label_" + validMetricLabel.ReplaceAllString(strings.ToLower(label), "_")
}
