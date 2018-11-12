package main

import (
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Returns a matrix of (version x operating system)
func getAzureScenarios() []testScenario {
	var scenarios []testScenario
	for _, v := range supportedVersions {
		// Ubuntu
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				Ubuntu: &kubermaticapiv1.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
				ContainerLinux: &kubermaticapiv1.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		//TODO: Fix
		//scenarios = append(scenarios, &azureScenario{
		//	version: v,
		//	nodeOsSpec: kubermaticapiv1.OperatingSystemSpec{
		//		CentOS: &kubermaticapiv1.CentOSSpec{},
		//	},
		//})
	}
	return scenarios
}

type azureScenario struct {
	version    *semver.Semver
	nodeOsSpec kubermaticapiv1.OperatingSystemSpec
}

func (s *azureScenario) Name() string {
	return fmt.Sprintf("azure-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *azureScenario) Cluster(secrets secrets) *v1.Cluster {
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
				DatacenterName: "azure-westeurope",
				Azure: &v1.AzureCloudSpec{
					ClientID:       secrets.Azure.ClientID,
					ClientSecret:   secrets.Azure.ClientSecret,
					SubscriptionID: secrets.Azure.SubscriptionID,
					TenantID:       secrets.Azure.TenantID,
				},
			},
		},
	}
}

func (s *azureScenario) Nodes(num int) []*kubermaticapiv1.Node {
	var nodes []*kubermaticapiv1.Node
	for i := 0; i < num; i++ {
		node := &kubermaticapiv1.Node{
			ObjectMeta: kubermaticapiv1.ObjectMeta{},
			Spec: kubermaticapiv1.NodeSpec{
				Cloud: kubermaticapiv1.NodeCloudSpec{
					Azure: &kubermaticapiv1.AzureNodeSpec{
						Size: "Standard_F1",
					},
				},
				Versions: kubermaticapiv1.NodeVersionInfo{
					Kubelet: s.version.String(),
				},
				OperatingSystem: s.nodeOsSpec,
			},
		}

		nodes = append(nodes, node)
	}

	return nodes
}
