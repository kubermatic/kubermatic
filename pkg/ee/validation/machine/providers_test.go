package machine_test

import (
	"context"
	"encoding/json"
	"testing"

	vmwareclouddirectortypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	"github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/ee/validation/machine"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockCtrlRuntimeClient is a mock implementation of ctrlruntimeclient.Client
type MockCtrlRuntimeClient struct {
	ctrlruntimeclient.Client
	// Add fields to simulate internal state or return values here
}

// Implement methods of ctrlruntimeclient.Client that your function calls

func TestGetVMwareCloudDirectorResourceRequirements(t *testing.T) {
	// Example test case for successful path
	testCases := []struct {
		name        string
		config      *types.Config
		expectedErr bool
	}{
		{
			name: "valid VMware configuration",
			config: &types.Config{
				CloudProvider:     types.CloudProviderVMwareCloudDirector,
				CloudProviderSpec: genFakeVMWareSpec(4, 2048, 20),
			},
			expectedErr: false,
		},
		{
			name: "Should fail with DiskSize not defined",
			config: &types.Config{
				CloudProvider:     types.CloudProviderVMwareCloudDirector,
				CloudProviderSpec: genFakeVMWareSpec(4, 2048, 0),
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockCtrlRuntimeClient{}
			_, err := machine.GetVMwareCloudDirectorResourceRequirements(context.Background(), mockClient, tc.config)
			if err != nil {
				if !tc.expectedErr {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if err == nil && tc.expectedErr {
				t.Fatal("expected error, got none")
			}
		})
	}
}

func genFakeVMWareSpec(cpu, ram, disk int64) runtime.RawExtension {
	var diskSize *int64

	if disk != 0 {
		diskSize = new(int64)
		*diskSize = disk
	}
	vmwareconfig := &vmwareclouddirectortypes.RawConfig{
		CPUs:       cpu,
		MemoryMB:   ram,
		DiskSizeGB: diskSize,
	}
	rawBytes, _ := json.Marshal(vmwareconfig)
	return runtime.RawExtension{
		Raw: rawBytes,
	}
}
