package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

const alibabaDC = "alibaba-eu-central-1a"

// Returns a matrix of (version x operating system)
func getAlibabaScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type alibabaScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *alibabaScenario) Name() string {
	return fmt.Sprintf("alibaba-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *alibabaScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: alibabaDC,
					Alibaba: &apimodels.AlibabaCloudSpec{
						AccessKeySecret: secrets.Alibaba.AccessKeySecret,
						AccessKeyID:     secrets.Alibaba.AccessKeyID,
					},
				},
				Version: s.version,
			},
		},
	}
}

func (s *alibabaScenario) NodeDeployments(num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	instanceType := "ecs.t1.xsmall"

	nds := []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Alibaba: &apimodels.AlibabaNodeSpec{
							InstanceType:             &instanceType,
							DiskSize:                 500,
							VSwitchID:                "",
							InternaalMaxBandwidthOut: 10,
							ZoneID: alibabaDC,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}

	// evenly distribute the nodes among deployments
	nodesInEachAZ := num / 3
	azsWithExtraNode := num % 3

	for i := range nds {
		var replicas int32
		if i < azsWithExtraNode {
			replicas = int32(nodesInEachAZ + 1)
		} else {
			replicas = int32(nodesInEachAZ)
		}
		nds[i].Spec.Replicas = &replicas
	}

	return nds, nil
}

func (s *alibabaScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
