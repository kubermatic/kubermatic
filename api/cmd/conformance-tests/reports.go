package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/onsi/ginkgo/reporters"
)

func collectReports(name, reportsDir string) (*reporters.JUnitTestSuite, error) {
	files, err := ioutil.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in reportsDir '%s': %v", reportsDir, err)
	}

	resultSuite := &reporters.JUnitTestSuite{Name: name}

	var individualReportFiles []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name(), "junit_") || !strings.HasSuffix(f.Name(), ".xml") {
			continue
		}

		absName := path.Join(reportsDir, f.Name())
		individualReportFiles = append(individualReportFiles, absName)

		b, err := ioutil.ReadFile(absName)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %v", absName, err)
		}

		suite := &reporters.JUnitTestSuite{}
		if err := xml.Unmarshal(b, suite); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report file '%s': %v", absName, err)
		}

		resultSuite = combineReports(name, resultSuite, suite)
	}

	for _, f := range individualReportFiles {
		if err := os.Remove(f); err != nil {
			return nil, fmt.Errorf("failed to remove report file: %v", err)
		}
	}

	return resultSuite, nil
}

func combineReports(name string, a, b *reporters.JUnitTestSuite) *reporters.JUnitTestSuite {
	resultSuite := &reporters.JUnitTestSuite{Name: name}

	resultSuite.Tests = a.Tests + b.Tests
	resultSuite.Errors += a.Errors + b.Errors
	resultSuite.Failures += a.Failures + b.Failures
	resultSuite.TestCases = append(a.TestCases, b.TestCases...)

	sort.Slice(resultSuite.TestCases, func(i, j int) bool { return resultSuite.TestCases[i].Name < resultSuite.TestCases[j].Name })

	return resultSuite
}

func printDetailedReport(report *reporters.JUnitTestSuite) {
	testBuf := &bytes.Buffer{}

	// Only print details errors in case we have a testcase which failed.
	// Printing everything which has an error will print the errors from retried tests as for each attempt a TestCase entry exists.
	if report.Failures > 0 || report.Errors > 0 {
		for _, t := range report.TestCases {
			if t.FailureMessage == nil {
				continue
			}

			fmt.Fprintln(testBuf, fmt.Sprintf("[FAIL] - %s", t.Name))
			fmt.Fprintln(testBuf, fmt.Sprintf("      %s", t.FailureMessage.Message))
		}
	}

	buf := &bytes.Buffer{}
	const separator = "============================================================="
	fmt.Fprintln(buf, separator)
	fmt.Fprintln(buf, fmt.Sprintf("Test results for: %s", report.Name))

	fmt.Fprint(buf, testBuf.String())

	fmt.Fprintln(buf, "----------------------------")
	fmt.Fprintln(buf, fmt.Sprintf("Passed: %d", report.Tests-report.Failures))
	fmt.Fprintln(buf, fmt.Sprintf("Failed: %d", report.Failures))
	fmt.Fprintln(buf, separator)

	fmt.Println(buf.String())
}
