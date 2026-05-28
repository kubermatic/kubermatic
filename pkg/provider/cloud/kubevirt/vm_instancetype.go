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

	"k8c.io/kubermatic/v2/pkg/provider"
	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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
	return describeInstanceType(ctx, client, it)
}

func describeInstanceType(ctx context.Context, client ctrlruntimeclient.Client, it *kubevirtv1.InstancetypeMatcher) (*provider.NodeCapacity, error) {
	switch it.Kind {
	case "VirtualMachineInstancetype": // namespaced: kubermatic standard or user-deployed custom
		if cap, err := describeNamespacedInstanceType(ctx, client, it.Name); cap != nil || err != nil {
			return cap, err
		}

	case "VirtualMachineClusterInstancetype": // cluster-wide
		if cap, err := describeClusterInstanceType(ctx, client, it.Name); cap != nil || err != nil {
			return cap, err
		}

	case "": // kind absent — saved before kind was added to the API; search both types
		if cap, err := describeClusterInstanceType(ctx, client, it.Name); cap != nil || err != nil {
			return cap, err
		}
		if cap, err := describeNamespacedInstanceType(ctx, client, it.Name); cap != nil || err != nil {
			return cap, err
		}
	}
	return nil, fmt.Errorf("VMI instancetype %s of Kind %s not found", it.Name, it.Kind)
}

func describeClusterInstanceType(ctx context.Context, client ctrlruntimeclient.Client, name string) (*provider.NodeCapacity, error) {
	clusterInstancetypes := kvinstancetypev1alpha1.VirtualMachineClusterInstancetypeList{}
	if err := client.List(ctx, &clusterInstancetypes); err != nil {
		return nil, err
	}
	for _, instancetype := range clusterInstancetypes.Items {
		if strings.EqualFold(instancetype.Name, name) {
			return instanceTypeToNodeCapacity(instancetype.Spec)
		}
	}
	return nil, nil
}

func describeNamespacedInstanceType(ctx context.Context, client ctrlruntimeclient.Client, name string) (*provider.NodeCapacity, error) {
	standardInstancetypes := GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})
	for _, instancetype := range standardInstancetypes {
		if strings.EqualFold(instancetype.Name, name) {
			return instanceTypeToNodeCapacity(instancetype.Spec)
		}
	}
	// Fall back to listing user-deployed namespaced VirtualMachineInstancetype objects
	// from the infra cluster — these are custom instancetypes (e.g. GPU variants)
	// that aren't part of the embedded Kubermatic standards.
	namespacedInstancetypes := kvinstancetypev1alpha1.VirtualMachineInstancetypeList{}
	if err := client.List(ctx, &namespacedInstancetypes); err != nil {
		return nil, err
	}
	for _, instancetype := range namespacedInstancetypes.Items {
		if strings.EqualFold(instancetype.Name, name) {
			return instanceTypeToNodeCapacity(instancetype.Spec)
		}
	}
	return nil, nil
}

// instanceTypeToNodeCapacity extracts cpu, mem and gpu resource requests from the kubevirt instancetype.
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
	if len(it.GPUs) > 0 {
		capacity.WithGPUCount(len(it.GPUs))
	}
	return capacity, nil
}
