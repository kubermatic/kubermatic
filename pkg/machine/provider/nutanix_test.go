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

package provider

import (
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/nutanix"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestNutanixConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewNutanixConfig().
		WithClusterName("clusterName").
		WithProjectName("projectName").
		WithSubnetName("subnetName").
		WithImageName("imageName").
		WithCPUs(1).
		WithMemoryMB(2).
		WithDiskSize(3).
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.ProjectName.Value != "projectName" {
		t.Fatal("Builder did not apply project name to the config.")
	}
}

type nutanixTestcase struct {
	baseTestcase[nutanix.RawConfig, kubermaticv1.DatacenterSpecNutanix]

	os providerconfig.OperatingSystem
}

func (tt *nutanixTestcase) Run(cluster *kubermaticv1.Cluster) (*nutanix.RawConfig, error) {
	return CompleteNutanixProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[nutanix.RawConfig] = &nutanixTestcase{}

func TestCompleteNutanixProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecNutanix{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteNutanixProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Nutanix = &kubermaticv1.NutanixCloudSpec{}
		if _, err := CompleteNutanixProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching Nutanix, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.NutanixCloudProvider),
		Nutanix:      &kubermaticv1.NutanixCloudSpec{},
	})

	defaultMachine := NewNutanixConfig().
		WithCategory("KKPCluster", fmt.Sprintf("kubernetes-%s", goodCluster.Name)).
		WithCategory("KKPProject", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey])

	testcases := []testcase[nutanix.RawConfig]{
		&nutanixTestcase{
			baseTestcase: baseTestcase[nutanix.RawConfig, kubermaticv1.DatacenterSpecNutanix]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecNutanix{
					AllowInsecure: true,
				},
				expected: cloneBuilder(defaultMachine).WithAllowInsecure(true),
			},
		},
		&nutanixTestcase{
			baseTestcase: baseTestcase[nutanix.RawConfig, kubermaticv1.DatacenterSpecNutanix]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecNutanix{
					AllowInsecure: false,
				},
				inputSpec: cloneBuilder(defaultMachine).WithAllowInsecure(true),
				expected:  cloneBuilder(defaultMachine).WithAllowInsecure(true),
			},
		},
		&nutanixTestcase{
			baseTestcase: baseTestcase[nutanix.RawConfig, kubermaticv1.DatacenterSpecNutanix]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecNutanix{
					Images: kubermaticv1.ImageList{
						providerconfig.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(defaultMachine).WithAllowInsecure(false).WithImageName("testimage"),
			},
			os: providerconfig.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
