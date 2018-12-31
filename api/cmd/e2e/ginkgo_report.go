package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func analyzeReport(prefix string, ctx *TestContext, t *testing.T) {
	files, err := ioutil.ReadDir(ctx.workingDir)
	if err != nil {
		t.Fatalf("failed to list files in from test directory '%s': %v", ctx.workingDir, err)
	}

	aggregatedReport := reporters.JUnitTestSuite{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name(), "junit_"+prefix) || !strings.HasSuffix(f.Name(), ".xml") {
			continue
		}

		absName := path.Join(ctx.workingDir, f.Name())
		b, err := ioutil.ReadFile(absName)
		if err != nil {
			t.Fatalf("failed to read file '%s': %v", absName, err)
		}

		suite := &reporters.JUnitTestSuite{}
		if err := xml.Unmarshal(b, suite); err != nil {
			t.Errorf("failed to unmarshal report file '%s': %v", absName, err)
		}

		os.Remove(absName)

		aggregatedReport.Tests += suite.Tests
		aggregatedReport.Errors += suite.Errors
		aggregatedReport.Failures += suite.Failures
		aggregatedReport.TestCases = append(aggregatedReport.TestCases, suite.TestCases...)
	}
	sort.Slice(aggregatedReport.TestCases, func(i, j int) bool { return aggregatedReport.TestCases[i].Name < aggregatedReport.TestCases[j].Name })

	for _, testCase := range aggregatedReport.TestCases {
		if testCase.Skipped != nil {
			continue
		}
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.FailureMessage != nil {
				t.Errorf("%s\n%s", testCase.FailureMessage.Type, testCase.FailureMessage.Message)
			}
		})
	}
}
