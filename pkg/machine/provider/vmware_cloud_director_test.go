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
	"k8c.io/machine-controller/sdk/cloudprovider/vmwareclouddirector"
	"k8c.io/machine-controller/sdk/providerconfig"
)

func TestVMwareCloudDirectorConfigBuilder(t *testing.T) {
	// call all With* functions once to ensure they all work...
	config := NewVMwareCloudDirectorConfig().
		WithCatalog("catalog").
		WithCPUs(1).
		WithCPUCores(2).
		WithMemoryMB(4096).
		WithDiskSizeGB(50).
		WithIPAllocationMode("mode").
		Build()

	// ... then randomly check whether the functions actually did anything
	if config.DiskSizeGB == nil || *config.DiskSizeGB != 50 {
		t.Fatal("Builder did not apply disk size to the config.")
	}
}

type vmwareclouddirectorTestcase struct {
	baseTestcase[vmwareclouddirector.RawConfig, kubermaticv1.DatacenterSpecVMwareCloudDirector]

	os providerconfig.OperatingSystem
}

func (tt *vmwareclouddirectorTestcase) Run(cluster *kubermaticv1.Cluster) (*vmwareclouddirector.RawConfig, error) {
	return CompleteVMwareCloudDirectorProviderSpec(tt.Input(), cluster, tt.datacenter, tt.os)
}

var _ testcase[vmwareclouddirector.RawConfig] = &vmwareclouddirectorTestcase{}

func TestCompleteVMwareCloudDirectorProviderSpec(t *testing.T) {
	t.Run("should validate the cluster's cloud provider", func(t *testing.T) {
		datacenter := &kubermaticv1.DatacenterSpecVMwareCloudDirector{}

		cluster := &kubermaticv1.Cluster{}
		if _, err := CompleteVMwareCloudDirectorProviderSpec(nil, cluster, datacenter, ""); err == nil {
			t.Error("Should have complained about invalid provider, but returned nil error.")
		}

		cluster.Spec.Cloud.VMwareCloudDirector = &kubermaticv1.VMwareCloudDirectorCloudSpec{}
		if _, err := CompleteVMwareCloudDirectorProviderSpec(nil, cluster, datacenter, ""); err != nil {
			t.Errorf("Cluster is now matching VMwareCloudDirector, should not have returned an error, but got: %v", err)
		}
	})

	goodCluster := genCluster(kubermaticv1.CloudSpec{
		ProviderName:        string(kubermaticv1.VMwareCloudDirectorCloudProvider),
		VMwareCloudDirector: &kubermaticv1.VMwareCloudDirectorCloudSpec{},
	})

	defaultMachine := NewVMwareCloudDirectorConfig().WithAllowInsecure(false)

	testcases := []testcase[vmwareclouddirector.RawConfig]{
		&vmwareclouddirectorTestcase{
			baseTestcase: baseTestcase[vmwareclouddirector.RawConfig, kubermaticv1.DatacenterSpecVMwareCloudDirector]{
				name: "should apply the values from the datacenter",
				datacenter: &kubermaticv1.DatacenterSpecVMwareCloudDirector{
					DefaultCatalog: "test-catalog",
				},
				expected: cloneBuilder(defaultMachine).WithCatalog("test-catalog"),
			},
		},
		&vmwareclouddirectorTestcase{
			baseTestcase: baseTestcase[vmwareclouddirector.RawConfig, kubermaticv1.DatacenterSpecVMwareCloudDirector]{
				name: "should not overwrite values in an existing spec",
				datacenter: &kubermaticv1.DatacenterSpecVMwareCloudDirector{
					DefaultCatalog: "test-catalog",
				},
				inputSpec: cloneBuilder(defaultMachine).WithCatalog("keep-me-vmwareclouddirector"),
				expected:  cloneBuilder(defaultMachine).WithCatalog("keep-me-vmwareclouddirector"),
			},
		},
		&vmwareclouddirectorTestcase{
			baseTestcase: baseTestcase[vmwareclouddirector.RawConfig, kubermaticv1.DatacenterSpecVMwareCloudDirector]{
				name: "should select the correct AMI based on the OS",
				datacenter: &kubermaticv1.DatacenterSpecVMwareCloudDirector{
					Templates: kubermaticv1.ImageList{
						providerconfig.OperatingSystemFlatcar: "testimage",
					},
				},
				expected: cloneBuilder(defaultMachine).WithTemplate("testimage"),
			},
			os: providerconfig.OperatingSystemFlatcar,
		},
	}

	runProviderTestcases(t, goodCluster, testcases)
}
