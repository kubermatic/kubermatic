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
	"k8c.io/machine-controller/sdk/cloudprovider/vsphere"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestVSphereConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewVSphereConfig().
		WithCPUs(1).
		WithMemoryMB(2).
		WithDiskSizeGB(3).
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.MemoryMB != 2 {
		t.Fatal("Builder did not apply memory to the config.")
	}
}

type vsphereTestcase struct {
	baseTestcase[vsphere.RawConfig, kubermaticv1.DatacenterSpecVSphere]

	os providerconfig.OperatingSystem
}

func (tt *vsphereTestcase) Run(cluster *kubermaticv1.Cluster) (*vsphere.RawConfig, error) {
	return CompleteVSphereProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[vsphere.RawConfig] = &vsphereTestcase{}

func TestCompleteVSphereProviderConfigBasics(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecVSphere{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteVSphereProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.VSphere = &kubermaticv1.VSphereCloudSpec{}
		if _, err := CompleteVSphereProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching VSphere, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName: string(kubermaticv1.VSphereCloudProvider),
		VSphere:      &kubermaticv1.VSphereCloudSpec{},
	})

	defaultMachine := NewVSphereConfig().WithAllowInsecure(false)
	goodMachine := cloneBuilder(defaultMachine).WithFolder("/testcluster")

	testcases := []testcase[vsphere.RawConfig]{
		&vsphereTestcase{
			baseTestcase: baseTestcase[vsphere.RawConfig, kubermaticv1.DatacenterSpecVSphere]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecVSphere{
					Datacenter:       "test-dc-vsphere",
					DefaultDatastore: "default-ds",
				},
				expected: cloneBuilder(goodMachine).WithDatacenter("test-dc-vsphere").WithDatastore("default-ds"),
			},
		},
		&vsphereTestcase{
			baseTestcase: baseTestcase[vsphere.RawConfig, kubermaticv1.DatacenterSpecVSphere]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecVSphere{
					Datacenter:       "test-dc-vsphere",
					DefaultDatastore: "default-ds",
				},
				inputSpec: cloneBuilder(defaultMachine).WithDatacenter("keep-me-vsphere").WithDatastore("keep-ds"),
				expected:  cloneBuilder(goodMachine).WithDatacenter("keep-me-vsphere").WithDatastore("keep-ds"),
			},
		},
		&vsphereTestcase{
			baseTestcase: baseTestcase[vsphere.RawConfig, kubermaticv1.DatacenterSpecVSphere]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecVSphere{
					Templates: kubermaticv1.ImageList{
						providerconfig.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(goodMachine).WithTemplateVMName("testimage"),
			},
			os: providerconfig.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
