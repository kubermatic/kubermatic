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
	"k8c.io/machine-controller/sdk/cloudprovider/alibaba"
)

func TestAlibabaConfigBuilder(t *testing.T) {
	config := NewAlibabaConfig().
		WithInstanceType("foo").
		WithDiskSize(42).
		WithDiskType("foo").
		WithVSwitchID("vswitch1").
		WithRegion("region").
		WithZone("zany").
		WithInternetMaxBandwidthOut(128).
		Build()

	if config.InstanceType.Value != "foo" {
		t.Fatal("Builder did not apply instance type to the config.")
	}
}

type alibabaTestcase struct {
	baseTestcase[alibaba.RawConfig, kubermaticv1.DatacenterSpecAlibaba]
}

func (tt *alibabaTestcase) Run(cluster *kubermaticv1.Cluster) (*alibaba.RawConfig, error) {
	return CompleteAlibabaProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[alibaba.RawConfig] = &alibabaTestcase{}

func TestCompleteAlibabaProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecAlibaba{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteAlibabaProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Alibaba = &kubermaticv1.AlibabaCloudSpec{}
		if _, err := CompleteAlibabaProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching Alibaba, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.AlibabaCloudProvider),
		Alibaba:      &kubermaticv1.AlibabaCloudSpec{},
	})

	defaultMachine := NewAlibabaConfig().
		WithDiskSize(alibabaDefaultDiskSize).
		WithDiskType(alibabaDefaultDiskType).
		WithInternetMaxBandwidthOut(alibabaDefaultInternetMaxBandwidthOut)

	testcases := []testcase[alibaba.RawConfig]{
		&alibabaTestcase{
			baseTestcase[alibaba.RawConfig, kubermaticv1.DatacenterSpecAlibaba]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecAlibaba{
					Region: "testregion",
				},
				expected: cloneBuilder(defaultMachine).WithRegion("testregion").WithZone("testregiona"),
			},
		},
		&alibabaTestcase{
			baseTestcase[alibaba.RawConfig, kubermaticv1.DatacenterSpecAlibaba]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecAlibaba{
					Region: "testregion",
				},
				inputSpec: cloneBuilder(defaultMachine).WithRegion("keep-me-alibaba"),
				expected:  cloneBuilder(defaultMachine).WithRegion("keep-me-alibaba").WithZone("keep-me-alibabaa"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
