/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package diff

import (
	"fmt"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/yaml"
)

func SemanticallyEqual(expected, actual interface{}) bool {
	return apiequality.Semantic.DeepEqual(expected, actual)
}

func ObjectDiff(expected, actual interface{}) string {
	expectedYAML, err := yaml.Marshal(expected)
	if err != nil {
		return fmt.Sprintf("<failed to encode expected object as YAML: %v>", err)
	}

	actualYAML, err := yaml.Marshal(actual)
	if err != nil {
		return fmt.Sprintf("<failed to encode actual object as YAML: %v>", err)
	}

	return StringDiff(string(expectedYAML), string(actualYAML))
}

func StringDiff(expected, actual string) string {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
	}

	unidifiedDiff, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Sprintf("<failed to generate diff: %v>", err)
	}

	return unidifiedDiff
}
