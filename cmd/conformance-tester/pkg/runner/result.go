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

package runner

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/reporters"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

type testResult struct {
	report   *reporters.JUnitTestSuite
	duration time.Duration
	err      error
	scenario scenarios.Scenario
	cluster  *kubermaticv1.Cluster
}

func (t *testResult) Passed() bool {
	if t.err != nil {
		return false
	}

	if t.report == nil {
		return false
	}

	if len(t.report.TestCases) == 0 {
		return false
	}

	if t.report.Errors > 0 || t.report.Failures > 0 {
		return false
	}

	return true
}

func printDetailedReport(report *reporters.JUnitTestSuite) {
	const separator = "============================================================="

	fmt.Println(separator)
	fmt.Printf("Test results for: %s\n", report.Name)

	// Only print details errors in case we have a testcase which failed.
	// Printing everything which has an error will print the errors from retried tests as for each attempt a TestCase entry exists.
	if report.Failures > 0 || report.Errors > 0 {
		for _, t := range report.TestCases {
			if t.FailureMessage == nil {
				continue
			}

			fmt.Printf("[FAIL] - %s\n", t.Name)
			fmt.Printf("         %s\n", t.FailureMessage.Message)
		}
	}

	fmt.Println("----------------------------")
	fmt.Printf("Passed: %d\n", report.Tests-report.Failures)
	fmt.Printf("Failed: %d\n", report.Failures)
	fmt.Println(separator)
}
