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
	"k8c.io/machine-controller/sdk/cloudprovider/digitalocean"
)

func TestDigitaloceanConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewDigitaloceanConfig().
		WithRegion("region").
		WithSize("size").
		WithIPv6(true).
		WithPrivateNetworking(true).
		WithBackups(true).
		WithMonitoring(true).
		WithIPv6(false).
		WithPrivateNetworking(false).
		WithBackups(false).
		WithMonitoring(false).
		WithTag("foo").
		WithTag("foo"). // try to add the same tag twice
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.Size.Value != "size" {
		t.Fatal("Builder did not apply droplet size to the config.")
	}
}

type digitaloceanTestcase struct {
	baseTestcase[digitalocean.RawConfig, kubermaticv1.DatacenterSpecDigitalocean]
}

func (tt *digitaloceanTestcase) Run(cluster *kubermaticv1.Cluster) (*digitalocean.RawConfig, error) {
	return CompleteDigitaloceanProviderSpec(tt.Input(), cluster, tt.datacenter)
}

var _ testcase[digitalocean.RawConfig] = &digitaloceanTestcase{}

func TestCompleteDigitaloceanProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecDigitalocean{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteDigitaloceanProviderSpec(nil, cluster, datacenter); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Digitalocean = &kubermaticv1.DigitaloceanCloudSpec{}
		if _, err := CompleteDigitaloceanProviderSpec(nil, cluster, datacenter); err != nil {
			t.Errorf("Cluster is now matching Digitalocean, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.DigitaloceanCloudProvider),
		Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{},
	})

	defaultMachine := NewDigitaloceanConfig().
		WithPrivateNetworking(true).
		WithTag("kubernetes").
		WithTag(fmt.Sprintf("kubernetes-cluster-%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system-cluster-%s", goodCluster.Name)).
		WithTag(fmt.Sprintf("system-project-%s", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey]))

	testcases := []testcase[digitalocean.RawConfig]{
		&digitaloceanTestcase{
			baseTestcase: baseTestcase[digitalocean.RawConfig, kubermaticv1.DatacenterSpecDigitalocean]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecDigitalocean{
					Region: "testregion-digitalocean",
				},
				expected: cloneBuilder(defaultMachine).WithRegion("testregion-digitalocean"),
			},
		},
		&digitaloceanTestcase{
			baseTestcase: baseTestcase[digitalocean.RawConfig, kubermaticv1.DatacenterSpecDigitalocean]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecDigitalocean{
					Region: "testregion-digitalocean",
				},
				inputSpec: cloneBuilder(defaultMachine).WithRegion("keep-me-digitalocean"),
				expected:  cloneBuilder(defaultMachine).WithRegion("keep-me-digitalocean"),
			},
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
