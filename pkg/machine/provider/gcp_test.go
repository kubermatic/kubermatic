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
	"k8c.io/machine-controller/sdk/cloudprovider/gce"
)

func TestGCPConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewGCPConfig().
		WithZone("zone").
		WithMachineType("machineType").
		WithDiskSize(50).
		WithDiskType("diskType").
		WithNetwork("network").
		WithPreemptible(true).
		WithPreemptible(false).
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.MachineType.Value != "machineType" {
		t.Fatal("Builder did not apply machine type to the config.")
	}
}

type gcpTestcase struct {
	baseTestcase[gce.RawConfig, kubermaticv1.DatacenterSpecGCP]
}

func (tt *gcpTestcase) Run(cluster *kubermaticv1.Cluster) (*gce.RawConfig, error) {
	return CompleteGCPProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[gce.RawConfig] = &gcpTestcase{}

func TestCompleteGCPProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecGCP{
			Region:       "testregion",
			ZoneSuffixes: []string{"a", "b"},
		}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteGCPProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.GCP = &kubermaticv1.GCPCloudSpec{}
		if _, err := CompleteGCPProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching GCP, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.GCPCloudProvider),
		GCP:          &kubermaticv1.GCPCloudSpec{},
	})

	defaultMachine := NewGCPConfig().
		WithRegional(false).
		WithMultiZone(false).
		WithAssignPublicIPAddress(true).
		WithTag(fmt.Sprintf("kubernetes-cluster-%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system-cluster-%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system-project-%s", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey]))

	testcases := []testcase[gce.RawConfig]{
		&gcpTestcase{
			baseTestcase: baseTestcase[gce.RawConfig, kubermaticv1.DatacenterSpecGCP]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecGCP{
					Region:       "testregion",
					ZoneSuffixes: []string{"a", "b"},
				},
				expected: cloneBuilder(defaultMachine).WithZone("testregion-a"),
			},
		},
		&gcpTestcase{
			baseTestcase: baseTestcase[gce.RawConfig, kubermaticv1.DatacenterSpecGCP]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecGCP{
					Region:       "testregion",
					ZoneSuffixes: []string{"a", "b"},
				},
				inputSpec: cloneBuilder(defaultMachine).WithZone("keep-me-gce"),
				expected:  cloneBuilder(defaultMachine).WithZone("keep-me-gce"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
