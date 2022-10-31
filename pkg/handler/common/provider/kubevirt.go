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

package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var NewKubeVirtClient = kubevirt.NewClient

// kubeVirtPresets returns the kubevirtv1.VirtualMachineInstancePreset from the `default` namespace, concatenated with Kubermatic standard presets.
func kubeVirtPresets(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (*kubevirtv1.VirtualMachineInstancePresetList, error) {
	// From `default` namespace.
	vmiPresets := &kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Add standard presets to the list.
	vmiPresets.Items = append(vmiPresets.Items, kubevirt.GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

	return vmiPresets, nil
}

func KubeVirtVMIPreset(ctx context.Context, kubeconfig, flavor string) (*kubevirtv1.VirtualMachineInstancePreset, error) {
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// KubeVirt presets concatenated with Kubermatic standards.
	vmiPresets, err := kubeVirtPresets(ctx, client, kubeconfig)
	if err != nil {
		return nil, err
	}

	for _, vmiPreset := range vmiPresets.Items {
		if strings.EqualFold(vmiPreset.Name, flavor) {
			return &vmiPreset, nil
		}
	}
	return nil, fmt.Errorf("KubeVirt VMI preset %q not found", flavor)
}

func isCPUSpecified(cpu *kubevirtv1.CPU) bool {
	return cpu != nil && (cpu.Cores != 0 || cpu.Threads != 0 || cpu.Sockets != 0)
}

// GetKubeVirtPresetResourceDetails extracts cpu and mem resource requests from the kubevirt preset
// for CPU, take the value by priority:
// - check if spec.cpu is set, if socket and threads are set then do the calculation, use that
// - if resource request is set, use that
// - if resource limit is set, use that
// for memory, take the value by priority:
// - if resource request is set, use that
// - if resource limit is set, use that.
func GetKubeVirtPresetResourceDetails(presetSpec kubevirtv1.VirtualMachineInstancePresetSpec) (resource.Quantity, resource.Quantity, error) {
	var err error
	// Get CPU
	cpuReq := resource.Quantity{}

	if isCPUSpecified(presetSpec.Domain.CPU) {
		if !presetSpec.Domain.Resources.Requests.Cpu().IsZero() || !presetSpec.Domain.Resources.Limits.Cpu().IsZero() {
			return resource.Quantity{}, resource.Quantity{}, errors.New("should not specify both spec.domain.cpu and spec.domain.resources.[requests/limits].cpu in VMIPreset")
		}
		cores := presetSpec.Domain.CPU.Cores
		if cores == 0 {
			cores = 1
		}
		// if threads and sockets are set, calculate VCPU
		threads := presetSpec.Domain.CPU.Threads
		if threads == 0 {
			threads = 1
		}
		sockets := presetSpec.Domain.CPU.Sockets
		if sockets == 0 {
			sockets = 1
		}

		cpuReq, err = resource.ParseQuantity(strconv.Itoa(int(cores * threads * sockets)))
		if err != nil {
			return resource.Quantity{}, resource.Quantity{}, fmt.Errorf("error parsing calculated KubeVirt VCPU: %w", err)
		}
	} else {
		if !presetSpec.Domain.Resources.Requests.Cpu().IsZero() {
			cpuReq = *presetSpec.Domain.Resources.Requests.Cpu()
		}
		if !presetSpec.Domain.Resources.Limits.Cpu().IsZero() {
			cpuReq = *presetSpec.Domain.Resources.Limits.Cpu()
		}
	}

	// get MEM
	memReq := resource.Quantity{}
	if presetSpec.Domain.Resources.Requests.Memory().IsZero() && presetSpec.Domain.Resources.Limits.Memory().IsZero() {
		return resource.Quantity{}, resource.Quantity{}, errors.New("spec.domain.resources.[requests/limits].memory must be set in VMIPreset")
	}
	if !presetSpec.Domain.Resources.Requests.Memory().IsZero() {
		memReq = *presetSpec.Domain.Resources.Requests.Memory()
	}
	if !presetSpec.Domain.Resources.Limits.Memory().IsZero() {
		memReq = *presetSpec.Domain.Resources.Limits.Memory()
	}

	return cpuReq, memReq, nil
}
