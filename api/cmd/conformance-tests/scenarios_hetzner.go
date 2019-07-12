package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getHetznerScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &hetznerScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &hetznerScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				CentOS: &kubermaticapiv1.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type hetznerScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *hetznerScenario) Name() string {
	return fmt.Sprintf("hetzner-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *hetznerScenario) Cluster(secrets secrets) *v1.Cluster {
	return &v1.Cluster{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.ClusterSpec{
			Version:           *s.version,
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
				DatacenterName: "hetzner-fsn1",
				Hetzner: &v1.HetznerCloudSpec{
					Token: secrets.Hetzner.Token,
				},
			},
		},
	}
}

func (s *hetznerScenario) APICluster(secrets secrets) *apimodels.CreateClusterSpec {
	return nil
}

func (s *hetznerScenario) Nodes(num int, _ secrets) *kubermaticapiv1.NodeDeployment {
	return &kubermaticapiv1.NodeDeployment{
		Spec: kubermaticapiv1.NodeDeploymentSpec{
			Replicas: int32(num),
			Template: kubermaticapiv1.NodeSpec{
				Cloud: kubermaticapiv1.NodeCloudSpec{
					Hetzner: &kubermaticapiv1.HetznerNodeSpec{
						Type: "cx31",
					},
				},
				Versions: kubermaticapiv1.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		},
	}
}
