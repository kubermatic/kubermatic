// +build ignore

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"testing"

	"github.com/go-test/deep"
)

func TestGetPossibleVMNetworks(t *testing.T) {
	tests := []struct {
		name                 string
		expectedNetworkInfos []NetworkInfo
	}{
		{
			name: "get all networks",
			expectedNetworkInfos: []NetworkInfo{
				{
					AbsolutePath: "/kubermatic-e2e/network/e2e-networks/subfolder/e2e-distributed-port-group",
					RelativePath: "e2e-networks/subfolder/e2e-distributed-port-group",
					Type:         "DistributedVirtualPortgroup",
					Name:         "e2e-distributed-port-group",
				},
				{
					AbsolutePath: "/kubermatic-e2e/network/e2e-networks/subfolder/e2e-distributed-switch-uplinks",
					RelativePath: "e2e-networks/subfolder/e2e-distributed-switch-uplinks",
					Type:         "DistributedVirtualPortgroup",
					Name:         "e2e-distributed-switch-uplinks",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			networkInfos, err := GetNetworks(getTestDC(), vSphereUsername, vSpherePassword, nil)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(test.expectedNetworkInfos, networkInfos); diff != nil {
				t.Errorf("Got network infos differ from expected ones. Diff: %v", diff)
			}
		})
	}
}
