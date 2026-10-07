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

package machine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/validation/machine"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	kubevirttypes "k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestResourceQuotaValidation(t *testing.T) {
	l := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	testCases := []struct {
		name        string
		machine     *clusterv1alpha1.Machine
		expectedErr bool
	}{
		{
			name:        "quota that fits should succeed",
			machine:     genFakeMachine("2", "2G", "10G"),
			expectedErr: false,
		},
		{
			name:        "should fail with CPU quota exceeded",
			machine:     genFakeMachine("50", "2G", "10G"),
			expectedErr: true,
		},
		{
			name:        "should fail with Memory quota exceeded",
			machine:     genFakeMachine("2", "50G", "10G"),
			expectedErr: true,
		},
		{
			name:        "should fail with Storage quota exceeded",
			machine:     genFakeMachine("2", "2G", "5000G"),
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := machine.ValidateQuota(context.Background(), l, nil, "", tc.machine, nil, genResourceQuota())
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

func TestAcceleratorResourceQuotaValidation(t *testing.T) {
	l := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	now := time.Now()

	tests := []struct {
		name          string
		machine       *clusterv1alpha1.Machine
		quota         *kubermaticv1.ResourceQuota
		mutate        func(*kubermaticv1.ResourceQuota)
		errorContains string
	}{
		{
			name:    "inactive accelerator quota remains unenforced",
			machine: genKubeVirtMachine(t, nil),
			quota:   genAcceleratorResourceQuota(t, now, "2", "1"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				delete(quota.Annotations, resources.AcceleratorAccountingEnabledAnnotation)
			},
		},
		{
			name:    "active empty accelerator quota does not gate Machine creation",
			machine: genKubeVirtMachine(t, nil),
			quota:   genAcceleratorResourceQuota(t, now, "2", "1"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Spec.Quota.Accelerators = nil
				quota.Status.GlobalAcceleratorAccounting = nil
				quota.Status.LocalAcceleratorAccounting = nil
			},
		},
		{
			name:    "non-KubeVirt Machine is not accelerator gated",
			machine: genFakeMachine("2", "2G", "10G"),
			quota:   genAcceleratorResourceQuota(t, now, "2", "1"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Status.GlobalAcceleratorAccounting = nil
				quota.Status.LocalAcceleratorAccounting = nil
			},
		},
		{
			name:    "ready usage at limit is admitted",
			machine: genKubeVirtMachine(t, corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")}),
			quota:   genAcceleratorResourceQuota(t, now, "3", "1"),
		},
		{
			name:          "request above limit is rejected",
			machine:       genKubeVirtMachine(t, corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")}),
			quota:         genAcceleratorResourceQuota(t, now, "2", "1"),
			errorContains: "would exceed current quota",
		},
		{
			name:          "explicit zero denies positive demand",
			machine:       genKubeVirtMachine(t, corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("1")}),
			quota:         genAcceleratorResourceQuota(t, now, "0", "0"),
			errorContains: "would exceed current quota",
		},
		{
			name:    "unconfigured footprint key is unconstrained",
			machine: genKubeVirtMachine(t, corev1.ResourceList{"nvidia.com/other": resource.MustParse("9")}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "1"),
		},
		{
			name:    "verified empty footprint consumes zero",
			machine: genKubeVirtMachine(t, corev1.ResourceList{}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "1"),
		},
		{
			name:    "missing global usage key is ready zero usage",
			machine: genKubeVirtMachine(t, corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("1")}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "0"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Status.GlobalUsage.Accelerators = nil
			},
		},
		{
			name:          "missing footprint fails closed",
			machine:       genKubeVirtMachine(t, nil),
			quota:         genAcceleratorResourceQuota(t, now, "1", "0"),
			errorContains: "missing trusted accelerator footprint",
		},
		{
			name: "malformed footprint fails closed",
			machine: func() *clusterv1alpha1.Machine {
				machine := genKubeVirtMachine(t, corev1.ResourceList{})
				machine.Annotations[accelerator.FootprintAnnotationKey] = "not-json"
				return machine
			}(),
			quota:         genAcceleratorResourceQuota(t, now, "1", "0"),
			errorContains: "footprint is invalid",
		},
		{
			name:    "missing global readiness fails closed",
			machine: genKubeVirtMachine(t, corev1.ResourceList{}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "0"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Status.GlobalAcceleratorAccounting = nil
			},
			errorContains: "not globally ready",
		},
		{
			name:    "stale global readiness fails closed",
			machine: genKubeVirtMachine(t, corev1.ResourceList{}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "0"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Status.GlobalAcceleratorAccounting.ObservedAt = metav1.NewTime(now.Add(-resources.AcceleratorAccountingHeartbeatTimeout - time.Second))
			},
			errorContains: "global report is stale",
		},
		{
			name:    "mismatched Seed revision fails closed",
			machine: genKubeVirtMachine(t, corev1.ResourceList{}),
			quota:   genAcceleratorResourceQuota(t, now, "1", "0"),
			mutate: func(quota *kubermaticv1.ResourceQuota) {
				quota.Status.LocalAcceleratorAccounting.ObservedAccountingRevision = "different"
			},
			errorContains: "Seed report does not match",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mutate != nil {
				tc.mutate(tc.quota)
			}
			err := machine.ValidateQuota(context.Background(), l, nil, "", tc.machine, nil, tc.quota)
			if tc.errorContains == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.errorContains) {
				t.Fatalf("expected error containing %q, got %v", tc.errorContains, err)
			}
		})
	}
}

func genFakeMachine(cpu, memory, storage string) *clusterv1alpha1.Machine {
	return generator.GenTestMachine("fake",
		fmt.Sprintf(`{"cloudProvider":"fake", "cloudProviderSpec":{"cpu":"%s","memory":"%s","storage":"%s"}}`, cpu, memory, storage),
		nil, nil)
}

func genKubeVirtMachine(t *testing.T, footprintResources corev1.ResourceList) *clusterv1alpha1.Machine {
	t.Helper()
	kubeVirtConfig := kubevirttypes.RawConfig{
		VirtualMachine: kubevirttypes.VirtualMachine{
			Template: kubevirttypes.Template{
				Memory: providerconfig.ConfigVarString{Value: "2G"},
				VCPUs:  kubevirttypes.VCPUs{Cores: 2},
				PrimaryDisk: kubevirttypes.PrimaryDisk{Disk: kubevirttypes.Disk{
					Size: providerconfig.ConfigVarString{Value: "10G"},
				}},
			},
		},
	}
	rawKubeVirtConfig, err := json.Marshal(kubeVirtConfig)
	if err != nil {
		t.Fatalf("failed to encode KubeVirt cloud provider config: %v", err)
	}
	config := providerconfig.Config{
		CloudProvider:     providerconfig.CloudProviderKubeVirt,
		CloudProviderSpec: runtime.RawExtension{Raw: rawKubeVirtConfig},
	}
	rawConfig, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to encode KubeVirt provider config: %v", err)
	}
	machine := generator.GenTestMachine("kubevirt", string(rawConfig), nil, nil)
	if footprintResources != nil {
		encoded, err := accelerator.Encode(accelerator.NewKubeVirtFootprint(footprintResources))
		if err != nil {
			t.Fatalf("failed to encode accelerator footprint: %v", err)
		}
		machine.Annotations = map[string]string{accelerator.FootprintAnnotationKey: encoded}
	}
	return machine
}

func genAcceleratorResourceQuota(t *testing.T, now time.Time, limit, used string) *kubermaticv1.ResourceQuota {
	t.Helper()
	quota := genResourceQuota()
	quota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}
	quota.Spec.Quota.Accelerators = []kubermaticv1.AcceleratorQuota{{
		Provider: accelerator.ProviderKubeVirt,
		Resources: corev1.ResourceList{
			"nvidia.com/gpu": resource.MustParse(limit),
		},
	}}
	quota.Status.GlobalUsage.Accelerators = []kubermaticv1.AcceleratorQuota{{
		Provider: accelerator.ProviderKubeVirt,
		Resources: corev1.ResourceList{
			"nvidia.com/gpu": resource.MustParse(used),
		},
	}}
	digest := kubermaticv1.AcceleratorQuotaDigestFor(quota.Spec.Quota.Accelerators)
	quota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ActivationPhase:            kubermaticv1.AcceleratorAccountingPhaseReady,
		ObservedAccountingRevision: "revision-1",
		ObservedQuotaDigest:        digest,
		ObservedAt:                 metav1.NewTime(now),
		Ready:                      true,
	}
	quota.Status.LocalAcceleratorAccounting = &kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus{
		ObservedAccountingRevision: "revision-1",
		ObservedQuotaDigest:        digest,
		ObservedAt:                 metav1.NewTime(now),
		Ready:                      true,
	}
	return quota
}

func genResourceQuota() *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Spec.Quota = *kubermaticv1.NewResourceDetails(resource.MustParse("50"), resource.MustParse("50G"), resource.MustParse("1000G"))
	rq.Status.GlobalUsage = *kubermaticv1.NewResourceDetails(resource.MustParse("3"), resource.MustParse("3G"), resource.MustParse("60G"))

	return rq
}
