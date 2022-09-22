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

func instancetypeCreator(instancetype *kvinstancetypev1alpha1.VirtualMachineInstancetype) reconciling.NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter {
	return func() (string, reconciling.KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator) {
		return instancetype.Name, func(it *kvinstancetypev1alpha1.VirtualMachineInstancetype) (*kvinstancetypev1alpha1.VirtualMachineInstancetype, error) {
			it.Labels = instancetype.Labels
			it.Spec = instancetype.Spec
			return it, nil
		}
	}
}

// reconcileInstancetypes reconciles the Kubermatic standard VirtualMachineInstancetype into the dedicated namespace.
func reconcileInstancetypes(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	instancetypes := &kvinstancetypev1alpha1.VirtualMachineInstancetypeList{}

	// add Kubermatic standards
	instancetypes.Items = append(instancetypes.Items, GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})...)

	for _, instancetype := range instancetypes.Items {
		instancetypeCreators := []reconciling.NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter{
			instancetypeCreator(&instancetype),
		}
		if err := reconciling.ReconcileKvInstancetypeV1alpha1VirtualMachineInstancetypes(ctx, instancetypeCreators, namespace, client); err != nil {
			return err
		}
	}

	return nil
}

// GetKubermaticStandardInstancetypes returns the Kubermatic standard VirtualMachineInstancetypes.
func GetKubermaticStandardInstancetypes(client ctrlruntimeclient.Client, getter kvmanifests.ManifestFSGetter) []kvinstancetypev1alpha1.VirtualMachineInstancetype {
	objs := kvmanifests.KubernetesFromYaml(client, getter)
	instancetypes := make([]kvinstancetypev1alpha1.VirtualMachineInstancetype, 0, len(objs))
	for _, obj := range objs {
		instancetypes = append(instancetypes, *obj.(*kvinstancetypev1alpha1.VirtualMachineInstancetype))
	}
	return instancetypes
}
