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
	"k8c.io/machine-controller/sdk/cloudprovider/equinixmetal"
)

func TestEquinixMetalConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewEquinixMetalConfig().
		WithInstanceType("instanceType").
		WithMetro("metro").
		WithFacility("facility").
		WithProjectID("projectID").
		WithBillingCycle("billingCycle").
		WithTag("foo").
		WithTag("foo"). // try to add the same tag twice
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.Metro.Value != "metro" {
		t.Fatal("Builder did not apply metro to the config.")
	}
}

type equinixmetalTestcase struct {
	baseTestcase[equinixmetal.RawConfig, kubermaticv1.DatacenterSpecPacket]
}

func (tt *equinixmetalTestcase) Run(cluster *kubermaticv1.Cluster) (*equinixmetal.RawConfig, error) {
	return CompleteEquinixMetalProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[equinixmetal.RawConfig] = &equinixmetalTestcase{}

func TestCompleteEquinixMetalProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecPacket{
			Metro: "testmetro",
		}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteEquinixMetalProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Packet = &kubermaticv1.PacketCloudSpec{}
		if _, err := CompleteEquinixMetalProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching EquinixMetal, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.PacketCloudProvider),
		Packet:       &kubermaticv1.PacketCloudSpec{},
	})

	defaultMachine := NewEquinixMetalConfig().
		WithTag("kubernetes").
		WithTag(fmt.Sprintf("kubernetes-cluster-%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system/cluster:%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system/project:%s", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey]))

	testcases := []testcase[equinixmetal.RawConfig]{
		&equinixmetalTestcase{
			baseTestcase: baseTestcase[equinixmetal.RawConfig, kubermaticv1.DatacenterSpecPacket]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecPacket{
					Metro: "testmetro",
				},
				expected: cloneBuilder(defaultMachine).WithMetro("testmetro"),
			},
		},
		&equinixmetalTestcase{
			baseTestcase: baseTestcase[equinixmetal.RawConfig, kubermaticv1.DatacenterSpecPacket]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecPacket{
					Metro: "testmetro",
				},
				inputSpec: cloneBuilder(defaultMachine).WithMetro("keep-me-equinixmetal"),
				expected:  cloneBuilder(defaultMachine).WithMetro("keep-me-equinixmetal"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
