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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/onsi/ginkgo/reporters"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ScenarioStatus string

const (
	ScenarioPassed  ScenarioStatus = "passed"
	ScenarioFailed  ScenarioStatus = "failed"
	ScenarioSkipped ScenarioStatus = "skipped"
)

type Results struct {
	Options   *types.Options
	Scenarios []ScenarioResult
}

func (r *Results) HasFailures() bool {
	for _, scenario := range r.Scenarios {
		if scenario.Status == ScenarioFailed {
			return true
		}
	}

	return false
}

func (r *Results) PrintSummary() {
	fmt.Println("")
	fmt.Println("========================== RESULT ===========================")
	fmt.Println("Parameters:")
	fmt.Printf("  KKP Version............: %s (%s)\n", r.Options.KubermaticConfiguration.Status.KubermaticVersion, r.Options.KubermaticConfiguration.Status.KubermaticEdition)
	fmt.Printf("  Name Prefix............: %q\n", r.Options.NamePrefix)
	fmt.Printf("  OSM Enabled............: %v\n", r.Options.OperatingSystemManagerEnabled)
	fmt.Printf("  Dualstack Enabled......: %v\n", r.Options.DualStackEnabled)
	fmt.Printf("  Konnectivity Enabled...: %v\n", r.Options.KonnectivityEnabled)
	fmt.Printf("  Cluster Updates Enabled: %v\n", r.Options.TestClusterUpdate)
	fmt.Printf("  Enabled Tests..........: %v\n", sets.List(r.Options.Tests))
	fmt.Printf("  Scenario Options.......: %v\n", sets.List(r.Options.ScenarioOptions))
	fmt.Println("")
	fmt.Println("Test results:")

	// sort results alphabetically
	sort.Slice(r.Scenarios, func(i, j int) bool {
		iname := r.Scenarios[i].scenarioName
		jname := r.Scenarios[j].scenarioName

		return iname < jname
	})

	for _, result := range r.Scenarios {
		var prefix string

		switch result.Status {
		case ScenarioPassed:
			prefix = " OK "
		case ScenarioFailed:
			prefix = "FAIL"
		case ScenarioSkipped:
			prefix = "SKIP"
		default:
			prefix = string(result.Status)
		}

		scenarioResultMsg := fmt.Sprintf("[%s] - %s", prefix, result.scenarioName)

		if r.Options.TestClusterUpdate && result.cluster != nil {
			scenarioResultMsg = fmt.Sprintf("%s (updated to %s)", scenarioResultMsg, result.cluster.Spec.Version)
		}

		scenarioResultMsg = fmt.Sprintf("%s (%s)", scenarioResultMsg, result.Duration.Round(time.Second))

		if result.Message != "" {
			scenarioResultMsg = fmt.Sprintf("%s: %v", scenarioResultMsg, result.Message)
		}

		fmt.Println(scenarioResultMsg)
	}
}

func (r *Results) PrintJUnitDetails() {
	for _, result := range r.Scenarios {
		result.PrintJUnitDetails()
	}
}

func MergeResults(previous *ResultsFile, current *Results) *Results {
	output := &Results{
		Options:   current.Options,
		Scenarios: previous.Results,
	}

	for _, currentScenarioResult := range current.Scenarios {
		found := false

		for j, previousResult := range output.Scenarios {
			// found a matching result from a previous run; update it with
			// the new test results
			if previousResult.Equals(currentScenarioResult) {
				// we only want to _improve_ test results, i.e. never go back
				// back from a successful scenario to one that failed due to a flake
				if currentScenarioResult.BetterThan(previousResult) {
					output.Scenarios[j] = currentScenarioResult
				}

				found = true
				break
			}
		}

		if !found {
			output.Scenarios = append(output.Scenarios, currentScenarioResult)
		}
	}

	return output
}

func (r *Results) WriteToFile(filename string) error {
	output := ResultsFile{
		Configuration: TestConfiguration{
			OSMEnabled:          r.Options.OperatingSystemManagerEnabled,
			DualstackEnabled:    r.Options.DualStackEnabled,
			KonnectivityEnabled: r.Options.KonnectivityEnabled,
			TestClusterUpdate:   r.Options.TestClusterUpdate,
			Tests:               sets.List(r.Options.Tests),
		},
		Results: r.Scenarios,
	}

	// sort results alphabetically
	sort.Slice(output.Results, func(i, j int) bool {
		iname := output.Results[i].scenarioName
		jname := output.Results[j].scenarioName

		return iname < jname
	})

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(output); err != nil {
		return err
	}

	return nil
}

type ScenarioResult struct {
	report       *reporters.JUnitTestSuite
	cluster      *kubermaticv1.Cluster
	scenarioName string

	CloudProvider     kubermaticv1.ProviderType      `json:"cloudProvider"`
	OperatingSystem   providerconfig.OperatingSystem `json:"operatingSystem"`
	ContainerRuntime  string                         `json:"containerRuntime"`
	KubernetesRelease string                         `json:"kubernetesRelease"`
	KubernetesVersion semver.Semver                  `json:"kubernetesVersion"`
	KubermaticVersion string                         `json:"kubermaticVersion"`
	Duration          time.Duration                  `json:"duration"`
	ClusterName       string                         `json:"clusterName"`
	Status            ScenarioStatus                 `json:"status"`
	Message           string                         `json:"message"`
}

func (sr *ScenarioResult) BetterThan(other ScenarioResult) bool {
	switch sr.Status {
	case ScenarioFailed:
		return other.Status == ScenarioSkipped || other.Status == ScenarioFailed
	case ScenarioSkipped:
		return other.Status == ScenarioSkipped
	case ScenarioPassed:
		return true
	}

	return false
}

func (sr *ScenarioResult) Equals(other ScenarioResult) bool {
	return true &&
		other.CloudProvider == sr.CloudProvider &&
		other.OperatingSystem == sr.OperatingSystem &&
		other.ContainerRuntime == sr.ContainerRuntime &&
		other.KubernetesVersion == sr.KubernetesVersion
}

func (sr *ScenarioResult) MatchesScenario(scenario scenarios.Scenario) bool {
	return true &&
		scenario.CloudProvider() == sr.CloudProvider &&
		scenario.OperatingSystem() == sr.OperatingSystem &&
		scenario.ContainerRuntime() == sr.ContainerRuntime &&
		scenario.ClusterVersion() == sr.KubernetesVersion
}

func (r *ScenarioResult) PrintJUnitDetails() {
	if r.report == nil {
		return
	}

	const separator = "============================================================="

	fmt.Println(separator)
	fmt.Printf("Test results for: %s\n", r.report.Name)

	// Only print details errors in case we have a testcase which failed.
	// Printing everything which has an error will print the errors from retried tests as for each attempt a TestCase entry exists.
	if r.report.Failures > 0 || r.report.Errors > 0 {
		for _, t := range r.report.TestCases {
			if t.FailureMessage == nil {
				continue
			}

			fmt.Printf("[FAIL] - %s\n", t.Name)
			fmt.Printf("         %s\n", t.FailureMessage.Message)
		}
	}

	fmt.Println("----------------------------")
	fmt.Printf("Passed: %d\n", r.report.Tests-r.report.Failures)
	fmt.Printf("Failed: %d\n", r.report.Failures)
	fmt.Println(separator)
}

type ResultsFile struct {
	Configuration TestConfiguration `json:"configuration"`
	Results       []ScenarioResult  `json:"results"`
}

type TestConfiguration struct {
	OSMEnabled          bool     `json:"osmEnabled"`
	DualstackEnabled    bool     `json:"dualstackEnabled"`
	KonnectivityEnabled bool     `json:"konnectivityEnabled"`
	TestClusterUpdate   bool     `json:"testClusterUpdate"`
	Tests               []string `json:"tests"`
}

func LoadResultsFile(filename string) (*ResultsFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	output := ResultsFile{}
	if err := json.NewDecoder(f).Decode(&output); err != nil {
		return nil, err
	}

	return &output, err
}
