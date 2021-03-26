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

package main

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/reporters"
)

// junitReporterWrapper is a convenience func to get junit results for a step
// It will create a report, append it to the passed in testsuite and propagate
// the error of the executor back up
// TODO: Should we add optional retrying here to limit the amount of wrappers we need?
func junitReporterWrapper(
	testCaseName string,
	report *reporters.JUnitTestSuite,
	executor func() error,
	extraErrOutputFn ...func() string,
) error {
	junitTestCase := reporters.JUnitTestCase{
		Name:      testCaseName,
		ClassName: testCaseName,
	}

	startTime := time.Now()
	err := executor()
	junitTestCase.Time = time.Since(startTime).Seconds()
	if err != nil {
		junitTestCase.FailureMessage = &reporters.JUnitFailureMessage{Message: err.Error()}
		report.Failures++
		for _, extraOut := range extraErrOutputFn {
			extraOutString := extraOut()
			err = fmt.Errorf("%v\n%s", err, extraOutString)
			junitTestCase.FailureMessage.Message += "\n" + extraOutString
		}
	}

	report.TestCases = append(report.TestCases, junitTestCase)
	report.Tests++

	return err
}
