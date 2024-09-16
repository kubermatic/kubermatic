//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package machine

import (
	"context"
	"encoding/json"
	"testing"

	vmwareclouddirectortypes "k8c.io/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	"k8c.io/machine-controller/pkg/providerconfig/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type MockCtrlRuntimeClient struct {
	ctrlruntimeclient.Client
}

func TestGetVMwareCloudDirectorResourceRequirements(t *testing.T) {
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
			_, err := getVMwareCloudDirectorResourceRequirements(context.Background(), mockClient, tc.config)
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
