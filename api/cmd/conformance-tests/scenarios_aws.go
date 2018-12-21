package main

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getAWSScenarios(es excludeSelector, versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		if _, ok := es.Distributions[providerconfig.OperatingSystemUbuntu]; !ok {
			scenarios = append(scenarios, &awsScenario{
				version: v,
				nodeOsSpec: v2.OperatingSystemSpec{
					Ubuntu: &v2.UbuntuSpec{},
				},
			})
		}
		// CoreOS
		if _, ok := es.Distributions[providerconfig.OperatingSystemCoreos]; !ok {
			scenarios = append(scenarios, &awsScenario{
				version: v,
				nodeOsSpec: v2.OperatingSystemSpec{
					ContainerLinux: &v2.ContainerLinuxSpec{
						// Otherwise the nodes restart directly after creation - bad for tests
						DisableAutoUpdate: true,
					},
				},
			})
		}
		// CentOS
		//TODO: Fix
		// if _, ok := es.Distributions[providerconfig.OperatingSystemCentos]; !ok {
		//scenarios = append(scenarios, &awsScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv2.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv2.CentOSSpec{},
		//	},
		//})
		//}
	}
	return scenarios
}

type awsScenario struct {
	version    *semver.Version
	nodeOsSpec v2.OperatingSystemSpec
}

func (s *awsScenario) Name() string {
	return fmt.Sprintf("aws-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *awsScenario) Cluster(secrets secrets) *v1.Cluster {
	return &v1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.ClusterSpec{
			Version:           s.version.String(),
			HumanReadableName: s.Name(),
			ClusterNetwork: v1.ClusterNetworkingConfig{
				Services: v1.NetworkRanges{
					CIDRBlocks: []string{"10.10.10.0/24"},
				},
				Pods: v1.NetworkRanges{
					CIDRBlocks: []string{"172.25.0.0/16"},
				},
				DNSDomain: "cluster.local",
			},
			Cloud: v1.CloudSpec{
				DatacenterName: "aws-eu-central-1a",
				AWS: &v1.AWSCloudSpec{
					SecretAccessKey: secrets.AWS.SecretAccessKey,
					AccessKeyID:     secrets.AWS.AccessKeyID,
				},
			},
		},
	}
}

func (s *awsScenario) Nodes(num int) []*v2.Node {
	var nodes []*v2.Node
	for i := 0; i < num; i++ {
		node := &v2.Node{
			Metadata: v2.ObjectMeta{},
			Spec: v2.NodeSpec{
				Cloud: v2.NodeCloudSpec{
					AWS: &v2.AWSNodeSpec{
						InstanceType: "t2.medium",
						VolumeType:   "gp2",
						VolumeSize:   100,
					},
				},
				Versions: v2.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		}

		nodes = append(nodes, node)
	}

	return nodes
}
