package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getBYOScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &byoScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
	}
	return scenarios
}

type byoScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *byoScenario) Name() string {
	return fmt.Sprintf("byo-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *byoScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "byo-kubernetes",
					Bringyourown: nil,
				},
				Version: s.version,
			},
		},
	}
}

func (s *byoScenario) NodeDeployments(num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	// NodeDeployments are not supported on BYO
	return nil, nil
}

func (s *byoScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
