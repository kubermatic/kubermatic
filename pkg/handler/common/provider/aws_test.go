/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package provider_test

import (
	"strings"
	"testing"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/common/provider"
)

func TestAWSSizeARMFiltering(t *testing.T) {
	tests := []struct {
		name                 string
		region               string
		resourceQuota        v1.MachineDeploymentVMResourceQuota
		unexpectedNamePrefix []string
	}{
		{
			name:          "test ARM filtering",
			region:        "eu-central-1",
			resourceQuota: genDefaultMachineDeploymentVMResourceQuota(),
			// Instance List for ARM:
			// a1.2xlarge
			// a1.4xlarge
			// a1.large
			// a1.medium
			// a1.metal
			// a1.xlarge
			// c6g.12xlarge
			// c6g.16xlarge
			// c6g.2xlarge
			// c6g.4xlarge
			// c6g.8xlarge
			// c6g.large
			// c6g.medium
			// c6g.metal
			// c6g.xlarge
			// m6g.12xlarge
			// m6g.16xlarge
			// m6g.2xlarge
			// m6g.4xlarge
			// m6g.8xlarge
			// m6g.large
			// m6g.medium
			// m6g.metal
			// m6g.xlarge
			// r6g.12xlarge
			// r6g.16xlarge
			// r6g.2xlarge
			// r6g.4xlarge
			// r6g.8xlarge
			// r6g.large
			// r6g.medium
			// r6g.metal
			// r6g.xlarge
			// t4g.2xlarge
			// t4g.large
			// t4g.medium
			// t4g.micro
			// t4g.nano
			// t4g.small
			// t4g.xlarge
			// ARM instances on AWS are marked with these prefixes
			unexpectedNamePrefix: []string{"t4g", "c6g", "r6g", "m6g", "a1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			awsSizeList, err := provider.AWSSizes(test.region, test.resourceQuota)
			if err != nil {
				t.Fatal(err)
			}

			for _, size := range awsSizeList {
				for _, prefix := range test.unexpectedNamePrefix {
					if strings.HasPrefix(size.Name, prefix) {
						t.Fatalf("Resulting list has an ARM instance %s with unexpected prefix %s", size.Name, prefix)
					}
				}
			}
		})
	}
}

func genDefaultMachineDeploymentVMResourceQuota() v1.MachineDeploymentVMResourceQuota {
	return v1.MachineDeploymentVMResourceQuota{
		MinCPU:    0,
		MaxCPU:    20,
		MinRAM:    0,
		MaxRAM:    64,
		EnableGPU: true,
	}
}
