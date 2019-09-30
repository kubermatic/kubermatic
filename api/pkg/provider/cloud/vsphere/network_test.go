// +build integration

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
			networkInfos, err := GetNetworks(getTestCloudSpec(), getTestDC())
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(test.expectedNetworkInfos, networkInfos); diff != nil {
				t.Errorf("Got network infos differ from expected ones. Diff: %v", diff)
			}
		})
	}
}
