//go:build integration

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
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/diff"
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
					AbsolutePath: "/dc-1/network/VM Network",
					RelativePath: "VM Network",
					Type:         "Network",
					Name:         "VM Network",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			networkInfos, err := GetNetworks(context.Background(), getTestDC(), vSphereUsername, vSpherePassword, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !diff.SemanticallyEqual(test.expectedNetworkInfos, networkInfos) {
				t.Fatalf("Got network infos differ from expected ones:\n%v", diff.ObjectDiff(test.expectedNetworkInfos, networkInfos))
			}
		})
	}
}
