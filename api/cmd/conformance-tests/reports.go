package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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

		appendReport(resultSuite, suite)
	}

	for _, f := range individualReportFiles {
		if err := os.Remove(f); err != nil {
			return nil, fmt.Errorf("failed to remove report file: %v", err)
		}
	}

	return resultSuite, nil
}

func appendReport(report, toAppend *reporters.JUnitTestSuite) {
	report.Tests += toAppend.Tests
	report.Errors += toAppend.Errors
	report.Failures += toAppend.Failures
	report.TestCases = append(report.TestCases, toAppend.TestCases...)
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
			fmt.Printf("      %s\n", t.FailureMessage.Message)
		}
	}

	fmt.Println("----------------------------")
	fmt.Printf("Passed: %d\n", report.Tests-report.Failures)
	fmt.Printf("Failed: %d\n", report.Failures)
	fmt.Println(separator)
}
