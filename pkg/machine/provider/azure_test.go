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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/azure"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestAzureConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewAzureConfig().
		WithLocation("location").
		WithResourceGroup("resourceGroup").
		WithVNetResourceGroup("vNetResourceGroup").
		WithVMSize("vmSize").
		WithVNetName("vNetName").
		WithSubnetName("subnetName").
		WithLoadBalancerSku("soadBalancerSku").
		WithRouteTableName("routeTableName").
		WithAvailabilitySet("availabilitySet").
		WithSecurityGroupName("securityGroupName").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.VMSize.Value != "vmSize" {
		t.Fatal("Builder did not apply VM size to the config.")
	}
}

type azureTestcase struct {
	baseTestcase[azure.RawConfig, kubermaticv1.DatacenterSpecAzure]

	os providerconfig.OperatingSystem
}

func (tt *azureTestcase) Run(cluster *kubermaticv1.Cluster) (*azure.RawConfig, error) {
	return CompleteAzureProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[azure.RawConfig] = &azureTestcase{}

func TestCompleteAzureProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecAzure{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteAzureProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Azure = &kubermaticv1.AzureCloudSpec{}
		if _, err := CompleteAzureProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching Azure, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.AzureCloudProvider),
		Azure: &kubermaticv1.AzureCloudSpec{
			VNetName: "vnet",
		},
	})

	defaultMachine := NewAzureConfig().
		WithTag("system-cluster", goodCluster.Name).
		WithTag("KubernetesCluster", goodCluster.Name).
		WithTag("system-project", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey])

	// good machine is the base machine, but with values from the cluster already applied
	goodMachine := cloneBuilder(defaultMachine).WithVNetName(goodCluster.Spec.Cloud.Azure.VNetName)

	testcases := []testcase[azure.RawConfig]{
		&azureTestcase{
			baseTestcase: baseTestcase[azure.RawConfig, kubermaticv1.DatacenterSpecAzure]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecAzure{
					Location: "testlocation-azure",
				},
				expected: cloneBuilder(goodMachine).
					WithLocation("testlocation-azure"),
			},
		},
		&azureTestcase{
			baseTestcase: baseTestcase[azure.RawConfig, kubermaticv1.DatacenterSpecAzure]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecAzure{
					Location: "testlocation-azure",
				},
				inputSpec: cloneBuilder(defaultMachine).WithLocation("keep-me-azure"),
				expected:  cloneBuilder(goodMachine).WithLocation("keep-me-azure"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
