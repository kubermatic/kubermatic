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

package gomegautil

import (
	"fmt"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	"k8c.io/kubermatic/v3/pkg/test/diff"
)

// SemanticallyEqual is like reflect.DeepEqual, but focused on semantic equality instead of memory equality.
// If objects differ a diff will be produced thanks to diff.ObjectDiff
// for more information about semantically equality please take a look at diff.SemanticallyEqual.
func SemanticallyEqual(expected any) types.GomegaMatcher {
	return &SemanticallyEqualMatcher{
		expected: expected,
	}
}

type SemanticallyEqualMatcher struct {
	expected any
}

func (matcher *SemanticallyEqualMatcher) Match(actual any) (success bool, err error) {
	return diff.SemanticallyEqual(matcher.expected, actual), nil
}

func (matcher *SemanticallyEqualMatcher) FailureMessage(actual any) (message string) {
	return fmt.Sprintf("Objects should be equal. Diff:\n%s", diff.ObjectDiff(matcher.expected, actual))
}

func (matcher *SemanticallyEqualMatcher) NegatedFailureMessage(actual any) (message string) {
	return format.Message(actual, "not to equal", matcher.expected)
}
