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

package diff

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/yaml"
)

func DeepEqual(expected, actual interface{}) bool {
	return reflect.DeepEqual(expected, actual)
}

func SemanticallyEqual(expected, actual interface{}) bool {
	return apiequality.Semantic.DeepEqual(expected, actual)
}

// these types are copied directly from apimachinery/pkg/util/sets

type ordered interface {
	integer | float | ~string
}

type integer interface {
	signed | unsigned
}

type float interface {
	~float32 | ~float64
}

type signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func SetDiff[T ordered](expected, actual sets.Set[T]) string {
	return ObjectDiff(sets.List(expected), sets.List(actual))
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
