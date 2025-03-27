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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/openstack"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestOpenstackConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewOpenstackConfig().
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

type openstackTestcase struct {
	baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenstack]

	os providerconfig.OperatingSystem
}

func (tt *openstackTestcase) Run(cluster *kubermaticv1.Cluster) (*openstack.RawConfig, error) {
	return CompleteOpenstackProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[openstack.RawConfig] = &openstackTestcase{}

func TestCompleteOpenstackProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecOpenstack{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteOpenstackProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.Openstack = &kubermaticv1.OpenstackCloudSpec{}
		if _, err := CompleteOpenstackProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching Openstack, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.OpenstackCloudProvider),
		Openstack:    &kubermaticv1.OpenstackCloudSpec{},
	})

	defaultMachine := NewOpenstackConfig().
		WithTrustDevicePath(false).
		WithTag("kubernetes-cluster", goodCluster.Name).
		WithTag("system-cluster", goodCluster.Name).
		WithTag("system-project", goodCluster.Labels[kubermaticv1.ProjectIDLabelKey])

	testcases := []testcase[openstack.RawConfig]{
		&openstackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenstack]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecOpenstack{
					Region: "testregion-openstack",
				},
				expected: cloneBuilder(defaultMachine).WithRegion("testregion-openstack"),
			},
		},
		&openstackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenstack]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecOpenstack{
					Region: "testregion-openstack",
				},
				inputSpec: cloneBuilder(defaultMachine).WithRegion("keep-me-openstack"),
				expected:  cloneBuilder(defaultMachine).WithRegion("keep-me-openstack"),
			},
		},
		&openstackTestcase{
			baseTestcase: baseTestcase[openstack.RawConfig, kubermaticv1.DatacenterSpecOpenstack]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecOpenstack{
					Images: kubermaticv1.ImageList{
						providerconfig.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(defaultMachine).WithImage("testimage"),
			},
			os: providerconfig.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
