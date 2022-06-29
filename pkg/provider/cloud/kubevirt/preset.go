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

	kubevirtv1 "kubevirt.io/api/core/v1"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	presets.Items = append(presets.Items, *GetKubermaticStandardPreset())

	for _, preset := range presets.Items {
		presetCreators := []reconciling.NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter{
			presetCreator(&preset),
		}
		if err := reconciling.EnsureNamedObjects(ctx, client, namespace, presetCreators); err != nil {
			return err
		}
	}

	return nil
}

// GetKubermaticStandardPreset returns a standard VirtualMachineInstancePreset with 2 CPUs and 8Gi of memory.
func GetKubermaticStandardPreset() *kubevirtv1.VirtualMachineInstancePreset {
	cpuQuantity, err := resource.ParseQuantity("2")
	if err != nil {
		return nil
	}
	memoryQuantity, err := resource.ParseQuantity("8Gi")
	if err != nil {
		return nil
	}
	resourceList := corev1.ResourceList{
		corev1.ResourceMemory: memoryQuantity,
		corev1.ResourceCPU:    cpuQuantity,
	}

	return &kubevirtv1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubermatic-standard",
		},
		Spec: kubevirtv1.VirtualMachineInstancePresetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"kubevirt.io/flavor": "kubermatic-standard"},
			},
			Domain: &kubevirtv1.DomainSpec{
				Resources: kubevirtv1.ResourceRequirements{
					Requests: resourceList,
					Limits:   resourceList,
				},
			},
		},
	}
}
