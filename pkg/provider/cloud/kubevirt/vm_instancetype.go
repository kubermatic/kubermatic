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
	"fmt"
	"strconv"
	"strings"

	kubevirtv1 "kubevirt.io/api/core/v1"
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	"k8c.io/kubermatic/v3/pkg/provider"
	kvmanifests "k8c.io/kubermatic/v3/pkg/provider/cloud/kubevirt/manifests"
	"k8c.io/kubermatic/v3/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func instancetypeReconciler(instancetype *kvinstancetypev1alpha1.VirtualMachineInstancetype) reconciling.NamedVirtualMachineInstancetypeReconcilerFactory {
	return func() (string, reconciling.VirtualMachineInstancetypeReconciler) {
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
		instancetypeReconcilers := []reconciling.NamedVirtualMachineInstancetypeReconcilerFactory{
			instancetypeReconciler(&instancetype),
		}
		if err := reconciling.ReconcileVirtualMachineInstancetypes(ctx, instancetypeReconcilers, namespace, client); err != nil {
			return err
		}
	}

	return nil
}

// GetKubermaticStandardInstancetypes returns the Kubermatic standard VirtualMachineInstancetypes.
func GetKubermaticStandardInstancetypes(client ctrlruntimeclient.Client, getter kvmanifests.ManifestFSGetter) []kvinstancetypev1alpha1.VirtualMachineInstancetype {
	objs := kvmanifests.RuntimeFromYaml(client, getter)
	instancetypes := make([]kvinstancetypev1alpha1.VirtualMachineInstancetype, 0, len(objs))
	for _, obj := range objs {
		instancetypes = append(instancetypes, *obj.(*kvinstancetypev1alpha1.VirtualMachineInstancetype))
	}
	return instancetypes
}

// DescribeInstanceType returns the NodeCapacity from the VirtualMachine instancetype.
func DescribeInstanceType(ctx context.Context, kubeconfig string, it *kubevirtv1.InstancetypeMatcher) (*provider.NodeCapacity, error) {
	client, err := NewClient(kubeconfig, ClientOptions{})
	if err != nil {
		return nil, err
	}

	switch it.Kind {
	case "VirtualMachineInstancetype": // "standard" instancetype
		standardInstancetypes := GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})
		for _, instancetype := range standardInstancetypes {
			if strings.EqualFold(instancetype.Name, it.Name) {
				return instanceTypeToNodeCapacity(instancetype.Spec)
			}
		}

	case "VirtualMachineClusterInstancetype": // "custom" instancetype (cluster-wide).
		customInstancetypes := kvinstancetypev1alpha1.VirtualMachineClusterInstancetypeList{}
		err := client.List(ctx, &customInstancetypes)
		if err != nil {
			return nil, err
		}
		for _, instancetype := range customInstancetypes.Items {
			if strings.EqualFold(instancetype.Name, it.Name) {
				return instanceTypeToNodeCapacity(instancetype.Spec)
			}
		}
	}
	return nil, fmt.Errorf("VMI instancetype %s of Kind %s not found", it.Name, it.Kind)
}

// instanceTypeToNodeCapacity extracts cpu and mem resource requests from the kubevirt instancetype.
func instanceTypeToNodeCapacity(it kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec) (*provider.NodeCapacity, error) {
	capacity := provider.NewNodeCapacity()

	// CPU and Memory are mandatory fields in instancetype
	if !it.Memory.Guest.IsZero() {
		capacity.Memory = &it.Memory.Guest
	}

	cpu, err := resource.ParseQuantity(strconv.Itoa(int(it.CPU.Guest)))
	if err != nil {
		return nil, fmt.Errorf("error parsing instancetype CPU: %w", err)
	}
	if !cpu.IsZero() {
		capacity.WithCPUCount(int(cpu.Value()))
	}
	return capacity, nil
}
