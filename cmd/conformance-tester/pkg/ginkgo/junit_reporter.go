package ginkgo

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/runner"
)

// JUnitTestSuite represents a JUnit XML test suite, which in our case contains a single test case.
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      float64         `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case in a JUnit report.
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
}

// JUnitFailure contains the details of a test failure.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Value   string `xml:",cdata"`
}

// JUnitSkipped contains the reason a test was skipped.
type JUnitSkipped struct {
	Message string `xml:"message,attr"`
}

type Failure struct {
	Message string
	Step    string
}

// JUnitReporter is a Ginkgo reporter that creates a JUnit XML file for each completed spec.
// This mimics the per-scenario reporting of the original `main.go`-based runner.
type JUnitReporter struct {
	name      string
	outputDir string
	suite     JUnitTestSuite
	specCases []JUnitTestCase
	result    runner.ScenarioResult
	failures  []Failure
}

// NewJUnitReporter creates a new JUnitReporter that writes files to the specified directory.
func NewJUnitReporter(outputDir string) *JUnitReporter {
	return &JUnitReporter{
		outputDir: outputDir,
		suite: JUnitTestSuite{
			TestCases: []JUnitTestCase{},
		},
		result: runner.ScenarioResult{},
	}
}

// SpecDidComplete is called by Ginkgo whenever a spec finishes.
func (r *JUnitReporter) SpecDidComplete(report types.SpecReport) {
	// We only generate reports for specs that have actually run.
	if report.State == types.SpecStatePending {
		return
	}

	r.suite = JUnitTestSuite{}

	// className := strings.Join(report.ContainerHierarchyTexts, " ")

	var systemOutBuilder strings.Builder
	systemOutBuilder.WriteString(report.CapturedGinkgoWriterOutput)

	r.suite.Name = report.ContainerHierarchyTexts[len(report.ContainerHierarchyTexts)-1]
	r.suite.XMLName = xml.Name{Local: r.suite.Name}

	if len(report.SpecEvents) > 0 {
		systemOutBuilder.WriteString("\n\nSpec Events:\n")
		for _, event := range report.SpecEvents {
			if event.SpecEventType != types.SpecEventByEnd {
				continue
			}
			r.suite.TestCases = append(r.suite.TestCases, JUnitTestCase{
				Name:      event.Message,
				ClassName: event.Message,
				Time:      event.Duration.Seconds(),
			})
		}
	}

	// r.suite.Tests++
	// r.suite.Time += testCase.Time

	// switch report.State {
	// case types.SpecStateFailed:
	// 	r.suite.Failures++
	// 	testCase.Failure = &JUnitFailure{
	// 		Message: report.Failure.Message,
	// 		Type:    report.Failure.FailureNodeType.String(),
	// 		Value:   fmt.Sprintf("%s\n%s", report.Failure.Location.String(), report.Failure.Location.FullStackTrace),
	// 	}
	// case types.SpecStateSkipped:
	// 	r.suite.Skipped++
	// 	testCase.Skipped = &JUnitSkipped{Message: report.Failure.Message}
	// case types.SpecStatePanicked:
	// 	r.suite.Errors++
	// 	testCase.Failure = &JUnitFailure{
	// 		Message: "Panic",
	// 		Type:    "Panic",
	// 		Value:   fmt.Sprintf("%s\n%s", report.Failure.Message, report.LeafNodeLocation.FullStackTrace),
	// 	}
	// }
}

// A V2 reporter is any object that implements one or more of the reporter methods.
func (r *JUnitReporter) SpecWillRun(report types.SpecReport) {}
func (r *JUnitReporter) BeforeSuite(report types.Report) {
	// r.suite.Name = report.SuiteDescription
}
func (r *JUnitReporter) AfterSuite(report types.SpecReport) {
	filename := fmt.Sprintf("junit.%s.xml", r.name)
	path := filepath.Join(r.outputDir, filename)

	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("Failed to create JUnit report file %q: %v\n", path, err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(xml.Header); err != nil {
		fmt.Printf("Failed to write XML header to %q: %v\n", path, err)
		return
	}

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")
	if err := encoder.Encode(r.suite); err != nil {
		fmt.Printf("Failed to encode JUnit report to %q: %v\n", path, err)
	}
}
