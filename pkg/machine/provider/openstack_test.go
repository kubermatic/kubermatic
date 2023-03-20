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
	"time"

	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
)

func TestOpenStackConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewOpenStackConfig().
		WithImage("image").
		WithFlavor("flavor").
		WithRegion("region").
		WithInstanceReadyCheckPeriod(5*time.Second).
		WithInstanceReadyCheckTimeout(2*time.Minute).
		WithTag("key", "value").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.Image.Value != "image" {
		t.Fatal("Builder did not apply image to the config.")
	}
}

type openStackTestcase struct {
	baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenStack]

	os kubermaticv1.OperatingSystem
}

func (tt *openStackTestcase) Run(cluster *kubermaticv1.Cluster) (*openstack.RawConfig, error) {
	return CompleteOpenStackProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[openstack.RawConfig] = &openStackTestcase{}

func TestCompleteOpenStackProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecOpenStack{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteOpenStackProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.OpenStack = &kubermaticv1.OpenStackCloudSpec{}
		if _, err := CompleteOpenStackProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching Openstack, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: kubermaticv1.CloudProviderOpenStack,
		OpenStack:    &kubermaticv1.OpenStackCloudSpec{},
	})

	defaultMachine := NewOpenStackConfig().
		WithTrustDevicePath(false).
		WithTag("kubernetes-cluster", goodCluster.Name).
		WithTag("system-cluster", goodCluster.Name).
		WithTag("system-project", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey])

	testcases := []testcase[openstack.RawConfig]{
		&openStackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenStack]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecOpenStack{
					Region: "testregion-openstack",
				},
				expected: cloneBuilder(defaultMachine).WithRegion("testregion-openstack"),
			},
		},
		&openStackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenStack]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecOpenStack{
					Region: "testregion-openstack",
				},
				inputSpec: cloneBuilder(defaultMachine).WithRegion("keep-me-openstack"),
				expected:  cloneBuilder(defaultMachine).WithRegion("keep-me-openstack"),
			},
		},
		&openStackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenStack]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecOpenStack{
					Images: kubermaticv1.ImageList{
						kubermaticv1.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(defaultMachine).WithImage("testimage"),
			},
			os: kubermaticv1.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
