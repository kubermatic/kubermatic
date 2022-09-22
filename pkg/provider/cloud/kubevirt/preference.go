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

	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func preferenceCreator(preference *kvinstancetypev1alpha1.VirtualMachinePreference) reconciling.NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter {
	return func() (string, reconciling.KvInstancetypeV1alpha1VirtualMachinePreferenceCreator) {
		return preference.Name, func(p *kvinstancetypev1alpha1.VirtualMachinePreference) (*kvinstancetypev1alpha1.VirtualMachinePreference, error) {
			p.Labels = preference.Labels
			p.Spec = preference.Spec
			return p, nil
		}
	}
}

// reconcilePreferences reconciles the Kubermatic standard VirtualMachinePreference into the dedicated namespace.
func reconcilePreferences(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	prefs := &kvinstancetypev1alpha1.VirtualMachinePreferenceList{}

	// add Kubermatic standards
	prefs.Items = append(prefs.Items, GetKubermaticStandardPreferences(client, &kvmanifests.StandardPreferenceGetter{})...)

	for _, pref := range prefs.Items {
		preferenceCreators := []reconciling.NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter{
			preferenceCreator(&pref),
		}
		if err := reconciling.ReconcileKvInstancetypeV1alpha1VirtualMachinePreferences(ctx, preferenceCreators, namespace, client); err != nil {
			return err
		}
	}

	return nil
}

// GetKubermaticStandardPreferences returns the Kubermatic standard VirtualMachinePreferences.
func GetKubermaticStandardPreferences(client ctrlruntimeclient.Client, getter kvmanifests.ManifestFSGetter) []kvinstancetypev1alpha1.VirtualMachinePreference {
	objs := kvmanifests.RuntimeFromYaml(client, getter)
	preferences := make([]kvinstancetypev1alpha1.VirtualMachinePreference, 0, len(objs))
	for _, obj := range objs {
		preferences = append(preferences, *obj.(*kvinstancetypev1alpha1.VirtualMachinePreference))
	}
	return preferences
}
