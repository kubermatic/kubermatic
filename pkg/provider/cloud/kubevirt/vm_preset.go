/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/provider"
	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func presetCreator(preset *kubevirtv1.VirtualMachineInstancePreset) reconciling.NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter {
	return func() (string, reconciling.KubeVirtV1VirtualMachineInstancePresetCreator) {
		return preset.Name, func(p *kubevirtv1.VirtualMachineInstancePreset) (*kubevirtv1.VirtualMachineInstancePreset, error) {
			p.Labels = preset.Labels
			p.Spec = preset.Spec
			return p, nil
		}
	}
}

// reconcilePresets reconciles the VirtualMachineInstancePresets from the `default` namespace to the dedicated one.
func reconcilePresets(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	presets := &kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, presets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return err
	}

	// add Kubermatic standards
	presets.Items = append(presets.Items, GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

	for _, preset := range presets.Items {
		presetCreators := []reconciling.NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter{
			presetCreator(&preset),
		}
		if err := reconciling.ReconcileKubeVirtV1VirtualMachineInstancePresets(ctx, presetCreators, namespace, client); err != nil {
			return err
		}
	}

	return nil
}

// GetKubermaticStandardPresets returns the Kubermatic standard VirtualMachineInstancePresets.
func GetKubermaticStandardPresets(client ctrlruntimeclient.Client, getter kvmanifests.ManifestFSGetter) []kubevirtv1.VirtualMachineInstancePreset {
	objs := kvmanifests.RuntimeFromYaml(client, getter)
	presets := make([]kubevirtv1.VirtualMachineInstancePreset, 0, len(objs))
	for _, obj := range objs {
		presets = append(presets, *obj.(*kubevirtv1.VirtualMachineInstancePreset))
	}
	return presets
}

// Deprecated: use DescribeInstanceType instead.
func DescribeFlavor(ctx context.Context, kubeconfig, flavor string) (*provider.NodeCapacity, error) {
	client, err := NewClient(kubeconfig, ClientOptions{})
	if err != nil {
		return nil, err
	}

	// KubeVirt presets
	vmiPresets := &kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Append the Kubermatic standards
	vmiPresets.Items = append(vmiPresets.Items, GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

	for _, vmiPreset := range vmiPresets.Items {
		if strings.EqualFold(vmiPreset.Name, flavor) {
			return vmiPresetToNodeCapacity(vmiPreset)
		}
	}

	return nil, fmt.Errorf("VMI flavor %q not found", flavor)
}

// vmiPresetToNodeCapacity extracts cpu and mem resource requests from the kubevirt preset.
// For CPU, take the value by priority:
// - check if spec.cpu is set, if socket and threads are set then do the calculation, use that
// - if resource request is set, use that
// - if resource limit is set, use that
// for memory, take the value by priority:
// - if resource request is set, use that
// - if resource limit is set, use that.
func vmiPresetToNodeCapacity(preset kubevirtv1.VirtualMachineInstancePreset) (*provider.NodeCapacity, error) {
	spec := preset.Spec
	resources := spec.Domain.Resources
	capacity := provider.NewNodeCapacity()

	// get CPU count
	if isCPUSpecified(spec.Domain.CPU) {
		if !spec.Domain.Resources.Requests.Cpu().IsZero() || !spec.Domain.Resources.Limits.Cpu().IsZero() {
			return nil, errors.New("should not specify both spec.domain.cpu and spec.domain.resources.[requests/limits].cpu in VMIPreset")
		}
		cores := spec.Domain.CPU.Cores
		if cores == 0 {
			cores = 1
		}
		// if threads and sockets are set, calculate VCPU
		threads := spec.Domain.CPU.Threads
		if threads == 0 {
			threads = 1
		}
		sockets := spec.Domain.CPU.Sockets
		if sockets == 0 {
			sockets = 1
		}

		capacity.WithCPUCount(int(cores * threads * sockets))
	} else {
		if cpu := resources.Requests.Cpu(); !cpu.IsZero() {
			capacity.WithCPUCount(int(cpu.Value()))
		}

		if cpu := resources.Limits.Cpu(); !cpu.IsZero() {
			capacity.WithCPUCount(int(cpu.Value()))
		}
	}

	// get memory
	if resources.Requests.Memory().IsZero() && resources.Limits.Memory().IsZero() {
		return nil, errors.New("resources.[requests/limits].memory must be set in VMIPreset")
	}

	if memory := resources.Requests.Memory(); !memory.IsZero() {
		capacity.Memory = memory
	}

	if memory := resources.Limits.Memory(); !memory.IsZero() {
		capacity.Memory = memory
	}

	return capacity, nil
}

func isCPUSpecified(cpu *kubevirtv1.CPU) bool {
	return cpu != nil && (cpu.Cores != 0 || cpu.Threads != 0 || cpu.Sockets != 0)
}
