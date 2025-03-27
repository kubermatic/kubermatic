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
	"k8c.io/machine-controller/sdk/cloudprovider/hetzner"
)

func TestHetznerConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewHetznerConfig().
		WithServerType("serverType").
		WithDatacenter("datacenter").
		WithImage("image").
		WithLocation("location").
		WithNetwork("network").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.ServerType.Value != "serverType" {
		t.Fatal("Builder did not apply server type to the config.")
	}
}

type hetznerTestcase struct {
	baseTestcase[hetzner.RawConfig, kubermaticv1.DatacenterSpecHetzner]
}

func (tt *hetznerTestcase) Run(cluster *kubermaticv1.Cluster) (*hetzner.RawConfig, error) {
	return CompleteHetznerProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[hetzner.RawConfig] = &hetznerTestcase{}

func TestCompleteHetznerProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecHetzner{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteHetznerProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Hetzner = &kubermaticv1.HetznerCloudSpec{}
		if _, err := CompleteHetznerProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching Hetzner, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.HetznerCloudProvider),
		Hetzner:      &kubermaticv1.HetznerCloudSpec{},
	})

	defaultMachine := NewHetznerConfig()

	testcases := []testcase[hetzner.RawConfig]{
		&hetznerTestcase{
			baseTestcase: baseTestcase[hetzner.RawConfig, kubermaticv1.DatacenterSpecHetzner]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecHetzner{
					Datacenter: "test-dc-hetzner",
				},
				expected: cloneBuilder(defaultMachine).WithDatacenter("test-dc-hetzner"),
			},
		},
		&hetznerTestcase{
			baseTestcase: baseTestcase[hetzner.RawConfig, kubermaticv1.DatacenterSpecHetzner]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecHetzner{
					Datacenter: "test-dc-hetzner",
				},
				inputSpec: cloneBuilder(defaultMachine).WithDatacenter("keep-me-hetzner"),
				expected:  cloneBuilder(defaultMachine).WithDatacenter("keep-me-hetzner"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
