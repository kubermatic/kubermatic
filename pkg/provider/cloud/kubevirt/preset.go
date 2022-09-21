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

	kvmanifests "k8c.io/kubermatic/v2/pkg/resources/cloudcontroller/kubevirtmanifests"
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
	objs := kvmanifests.KubernetesFromYaml(client, getter)
	presets := make([]kubevirtv1.VirtualMachineInstancePreset, 0, len(objs))
	for _, obj := range objs {
		presets = append(presets, *obj.(*kubevirtv1.VirtualMachineInstancePreset))
	}
	return presets
}
