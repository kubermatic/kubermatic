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
	"time"

	"github.com/onsi/ginkgo/reporters"
)

func collectReports(name, reportsDir string, time time.Duration) (*reporters.JUnitTestSuite, error) {
	files, err := ioutil.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in reportsDir '%s': %v", reportsDir, err)
	}

	resultSuite := reporters.JUnitTestSuite{
		Time: time.Seconds(),
	}
	resultSuite.Name = name

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

		resultSuite.Tests += suite.Tests
		resultSuite.Errors += suite.Errors
		resultSuite.Failures += suite.Failures
		resultSuite.TestCases = append(resultSuite.TestCases, suite.TestCases...)
	}
	sort.Slice(resultSuite.TestCases, func(i, j int) bool { return resultSuite.TestCases[i].Name < resultSuite.TestCases[j].Name })

	b, err := xml.Marshal(resultSuite)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal combined report file: %v", err)
	}

	if err := ioutil.WriteFile(path.Join(reportsDir, "junit.xml"), b, 0644); err != nil {
		return nil, fmt.Errorf("failed to wrte combined report file: %v", err)
	}

	for _, f := range individualReportFiles {
		if err := os.Remove(f); err != nil {
			return nil, fmt.Errorf("failed to remove report file: %v", err)
		}
	}

	return &resultSuite, nil
}

func printDetailedReport(report *reporters.JUnitTestSuite) {
	testBuf := &bytes.Buffer{}
	for _, t := range report.TestCases {
		status := "PASS"
		if t.FailureMessage != nil {
			status = "FAIL"
		}

		if t.FailureMessage == nil {
			continue
		}

		fmt.Fprintln(testBuf, fmt.Sprintf("[%s] - %s", status, t.Name))
		if t.FailureMessage != nil {
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
