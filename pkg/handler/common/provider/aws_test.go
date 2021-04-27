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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/common/provider"
)

func TestAWSSize(t *testing.T) {
	tests := []struct {
		name          string
		region        string
		resourceQuota v1.MachineDeploymentVMResourceQuota
		expectedNames []string
	}{
		{
			name:          "test ARM filtering",
			region:        "eu-central-1",
			resourceQuota: genDefaultMachineDeploymentVMResourceQuota(),
			// t4g.small is a instance with ARM which is filtered out here
			expectedNames: []string{"t3a.small", "t3.small"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			awsSizeList, err := provider.AWSSizes(test.region, test.resourceQuota)
			if err != nil {
				t.Fatal(err)
			}

			var resultNames []string
			for _, size := range awsSizeList {
				resultNames = append(resultNames, size.Name)
			}

			if !reflect.DeepEqual(resultNames, test.expectedNames) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(resultNames, test.expectedNames))
			}
		})
	}
}

func genDefaultMachineDeploymentVMResourceQuota() v1.MachineDeploymentVMResourceQuota {
	return v1.MachineDeploymentVMResourceQuota{
		MinCPU:    2,
		MaxCPU:    2,
		MinRAM:    2,
		MaxRAM:    2,
		EnableGPU: false,
	}
}
