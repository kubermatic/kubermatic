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
	"k8c.io/machine-controller/sdk/cloudprovider/anexia"
)

func TestAnexiaConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewAnexiaConfig().
		WithLocationID("foo").
		WithTemplateID("foo").
		WithVlanID("vlan1").
		WithCPUs(2).
		WithMemory(4096).
		WithDiskSize(42).
		AddDisk(2, "type").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.LocationID.Value != "foo" {
		t.Fatal("Builder did not apply location ID to the config.")
	}
}

type anexiaTestcase struct {
	baseTestcase[anexia.RawConfig, kubermaticv1.DatacenterSpecAnexia]
}

func (tt *anexiaTestcase) Run(cluster *kubermaticv1.Cluster) (*anexia.RawConfig, error) {
	return CompleteAnexiaProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[anexia.RawConfig] = &anexiaTestcase{}

func TestCompleteAnexiaProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecAnexia{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteAnexiaProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Anexia = &kubermaticv1.AnexiaCloudSpec{}
		if _, err := CompleteAnexiaProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching Anexia, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.AnexiaCloudProvider),
		Anexia:       &kubermaticv1.AnexiaCloudSpec{},
	})

	defaultMachine := NewAnexiaConfig()

	testcases := []testcase[anexia.RawConfig]{
		&anexiaTestcase{
			baseTestcase[anexia.RawConfig, kubermaticv1.DatacenterSpecAnexia]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecAnexia{
					LocationID: "testlocation",
				},
				expected: cloneBuilder(defaultMachine).WithLocationID("testlocation"),
			},
		},
		&anexiaTestcase{
			baseTestcase[anexia.RawConfig, kubermaticv1.DatacenterSpecAnexia]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecAnexia{
					LocationID: "testlocation",
				},
				inputSpec: cloneBuilder(defaultMachine).WithLocationID("keep-me-anexia"),
				expected:  cloneBuilder(defaultMachine).WithLocationID("keep-me-anexia"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
