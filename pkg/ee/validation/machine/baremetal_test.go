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
	"reflect"
	"testing"

	"k8c.io/machine-controller/sdk/cloudprovider/baremetal"
	"k8c.io/machine-controller/sdk/cloudprovider/baremetal/plugins/tinkerbell"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var hardwareGVK = schema.GroupVersionKind{
	Group:   hardwareGVKGroup,
	Version: hardwareGVKVersion,
	Kind:    hardwareGVKKind,
}

func hardwareObject(name, namespace, annotation string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(hardwareGVK)
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetAnnotations(map[string]string{agentAttributesAnnotation: annotation})
	return obj
}

func TestResourceDetailsFromHardware(t *testing.T) {
	testCases := []struct {
		name          string
		hardware      *unstructured.Unstructured
		expectCPU     string // expected quantity string ("" means expect 0 / not asserted)
		expectMem     string
		expectStorage string
		expectErr     bool
	}{
		{
			name:          "hardware with full agent-attributes",
			hardware:      hardwareObject("hw1", "default", `{"cpu":{"totalCores":8},"memory":{"total":"96Gi"},"blockDevices":[{"size":"4Ti"},{"size":"512Gi"}]}`),
			expectCPU:     "8",
			expectMem:     "96Gi",
			expectStorage: "4608Gi", // 4Ti + 512Gi, normalized
			expectErr:     false,
		},
		{
			name:          "hardware with no annotation",
			hardware:      hardwareObject("hw2", "default", ""),
			expectCPU:     "0",
			expectMem:     "0",
			expectStorage: "0",
			expectErr:     true,
		},
		{
			name:          "hardware with malformed annotation",
			hardware:      hardwareObject("hw3", "default", `{not valid json`),
			expectCPU:     "0",
			expectMem:     "0",
			expectStorage: "0",
			expectErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			details, err := resourceDetailsFromHardware(tc.hardware)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got none: %+v", details)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectCPU != "" {
				if got := details.CPU().String(); got != tc.expectCPU {
					t.Errorf("CPU: got %q, want %q", got, tc.expectCPU)
				}
			}
			if tc.expectMem != "" {
				if got := details.Memory().String(); got != tc.expectMem {
					t.Errorf("Memory: got %q, want %q", got, tc.expectMem)
				}
			}
			if tc.expectStorage != "" {
				if got := details.Storage().String(); got != tc.expectStorage {
					t.Errorf("Storage: got %q, want %q", got, tc.expectStorage)
				}
			}
		})
	}
}

func TestResourceDetailsFromHardwareSpecResourcesFallback(t *testing.T) {
	obj := hardwareObject("hw-res", "default", "")
	// annotate with empty string but provide spec.resources fallback
	obj.SetAnnotations(nil)
	obj.Object["spec"] = map[string]interface{}{
		"resources": map[string]interface{}{
			"cpu":    "4",
			"memory": "16Gi",
			"disk":   "500Gi",
		},
	}

	details, err := resourceDetailsFromHardware(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := details.CPU().String(); got != "4" {
		t.Errorf("CPU: got %q, want %q", got, "4")
	}
	if got := details.Memory().String(); got != "16Gi" {
		t.Errorf("Memory: got %q, want %q", got, "16Gi")
	}
	if got := details.Storage().String(); got != "500Gi" {
		t.Errorf("Storage: got %q, want %q", got, "500Gi")
	}
}

func TestGetBaremetalResourceDetailsFromCluster(t *testing.T) {
	testCases := []struct {
		name      string
		objects   []*unstructured.Unstructured
		ref       types.NamespacedName
		expectCPU string
		expectErr bool
	}{
		{
			name: "hardware found with agent-attributes",
			objects: []*unstructured.Unstructured{
				hardwareObject("hw1", "default", `{"cpu":{"totalCores":8},"memory":{"total":"96Gi"},"blockDevices":[{"size":"4Ti"}]}`),
			},
			ref:       types.NamespacedName{Namespace: "default", Name: "hw1"},
			expectCPU: "8",
			expectErr: false,
		},
		{
			name:      "hardware not found",
			objects:   nil,
			ref:       types.NamespacedName{Namespace: "default", Name: "missing"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := ctrlruntimefakeclient.NewClientBuilder()
			for _, obj := range tc.objects {
				builder = builder.WithObjects(obj)
			}
			fakeClient := builder.Build()

			details, err := getBaremetalResourceDetailsFromCluster(context.Background(), fakeClient, tc.ref)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got none: %+v", details)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectCPU != "" {
				if got := details.CPU().String(); got != tc.expectCPU {
					t.Errorf("CPU: got %q, want %q", got, tc.expectCPU)
				}
			}
		})
	}
}

func TestGetBaremetalResourceRequirementsReturnsZeroOnFailure(t *testing.T) {
	// A nil-embedded mock client (like providers_test.go's MockCtrlRuntimeClient) never
	// connects anywhere; with no inline kubeconfig and no TINK_KUBECONFIG the resolver
	// returns zero deterministically.
	mockClient := &MockCtrlRuntimeClient{}
	config := &providerconfig.Config{
		CloudProvider:     providerconfig.CloudProviderBaremetal,
		CloudProviderSpec: genFakeBaremetalSpec(""),
	}

	details, err := getBaremetalResourceRequirements(context.Background(), mockClient, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details == nil {
		t.Fatal("expected non-nil ResourceDetails")
	}
	zero := resource.Quantity{}
	if !reflect.DeepEqual(*details.CPU(), zero) ||
		!reflect.DeepEqual(*details.Memory(), zero) ||
		!reflect.DeepEqual(*details.Storage(), zero) {
		t.Fatalf("expected zero resource details, got %+v", details)
	}
}

// genFakeBaremetalSpec builds a baremetal provider config. kubeconfig may be empty to force the
// deterministic zero-fallback path.
func genFakeBaremetalSpec(kubeconfig string) runtime.RawExtension {
	driverSpec := tinkerbell.TinkerbellPluginSpec{
		ClusterName: providerconfig.ConfigVarString{Value: "cluster"},
		Auth: tinkerbell.Auth{
			Kubeconfig: providerconfig.ConfigVarString{Value: kubeconfig},
		},
		OSImageURL:  providerconfig.ConfigVarString{Value: "http://example.com/os.img"},
		HardwareRef: types.NamespacedName{Namespace: "default", Name: "hw1"},
	}
	driverSpecRaw, _ := json.Marshal(driverSpec)

	rawConfig := &baremetal.RawConfig{
		Driver:     providerconfig.ConfigVarString{Value: "tinkerbell"},
		DriverSpec: runtime.RawExtension{Raw: driverSpecRaw},
	}
	rawBytes, _ := json.Marshal(rawConfig)
	return runtime.RawExtension{Raw: rawBytes}
}
