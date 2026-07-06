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
	kvinstancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"k8c.io/kubermatic/v2/pkg/provider"
	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func instancetypeReconciler(instancetype *kvinstancetypev1beta1.VirtualMachineInstancetype) reconciling.NamedVirtualMachineInstancetypeReconcilerFactory {
	return func() (string, reconciling.VirtualMachineInstancetypeReconciler) {
		return instancetype.Name, func(it *kvinstancetypev1beta1.VirtualMachineInstancetype) (*kvinstancetypev1beta1.VirtualMachineInstancetype, error) {
			it.Labels = instancetype.Labels
			it.Spec = instancetype.Spec
			return it, nil
		}
	}
}

// reconcileInstancetypes reconciles the Kubermatic standard VirtualMachineInstancetype into the dedicated namespace.
func reconcileInstancetypes(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	instancetypes := &kvinstancetypev1beta1.VirtualMachineInstancetypeList{}

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
func GetKubermaticStandardInstancetypes(client ctrlruntimeclient.Client, getter kvmanifests.ManifestFSGetter) []kvinstancetypev1beta1.VirtualMachineInstancetype {
	objs := kvmanifests.RuntimeFromYaml(client, getter)
	instancetypes := make([]kvinstancetypev1beta1.VirtualMachineInstancetype, 0, len(objs))
	for _, obj := range objs {
		instancetypes = append(instancetypes, *obj.(*kvinstancetypev1beta1.VirtualMachineInstancetype))
	}
	return instancetypes
}

// DescribeInstanceType returns the NodeCapacity from the VirtualMachine instancetype.
// namespace is the KubeVirt infra-cluster namespace that holds the cluster's namespaced
// instancetypes. Resolving a custom (non-standard) namespaced instancetype without it fails,
// to avoid an unscoped cross-tenant lookup. The embedded Kubermatic standard instancetypes are
// namespace-independent and still resolve when it is empty.
func DescribeInstanceType(ctx context.Context, kubeconfig string, namespace string, it *kubevirtv1.InstancetypeMatcher) (*provider.NodeCapacity, error) {
	client, err := NewClient(kubeconfig, ClientOptions{})
	if err != nil {
		return nil, err
	}
	return describeInstanceType(ctx, client, namespace, it)
}

func describeInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace string, it *kubevirtv1.InstancetypeMatcher) (*provider.NodeCapacity, error) {
	switch it.Kind {
	case "VirtualMachineInstancetype": // namespaced: kubermatic standard or user-deployed custom
		if nodeCap, err := describeNamespacedInstanceType(ctx, client, namespace, it.Name); nodeCap != nil || err != nil {
			return nodeCap, err
		}

	case "VirtualMachineClusterInstancetype": // cluster-wide
		if nodeCap, err := describeClusterInstanceType(ctx, client, it.Name); nodeCap != nil || err != nil {
			return nodeCap, err
		}

	case "": // kind absent — saved before kind was added to the API; search both types
		if nodeCap, err := describeClusterInstanceType(ctx, client, it.Name); nodeCap != nil || err != nil {
			return nodeCap, err
		}
		if nodeCap, err := describeNamespacedInstanceType(ctx, client, namespace, it.Name); nodeCap != nil || err != nil {
			return nodeCap, err
		}
	}
	return nil, fmt.Errorf("VMI instancetype %s of Kind %s not found", it.Name, it.Kind)
}

func describeClusterInstanceType(ctx context.Context, client ctrlruntimeclient.Client, name string) (*provider.NodeCapacity, error) {
	clusterInstancetypes := kvinstancetypev1beta1.VirtualMachineClusterInstancetypeList{}
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

func describeNamespacedInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) (*provider.NodeCapacity, error) {
	standardInstancetypes := GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})
	for _, instancetype := range standardInstancetypes {
		if strings.EqualFold(instancetype.Name, name) {
			return instanceTypeToNodeCapacity(instancetype.Spec)
		}
	}
	// Fall back to listing user-deployed namespaced VirtualMachineInstancetype objects
	// from the infra cluster — these are custom instancetypes (e.g. GPU variants)
	// that aren't part of the embedded Kubermatic standards.
	//
	// This requires the cluster's infra namespace: a custom instancetype must be resolved
	// against the tenant that owns it, because two tenants may define instancetypes with the
	// same name but different specs. Without a namespace we cannot do that safely, so we fail
	// rather than risk feeding another tenant's spec into resource-quota validation.
	if namespace == "" {
		return nil, fmt.Errorf("cannot resolve namespaced instancetype %q: no KubeVirt infra namespace configured", name)
	}
	namespacedInstancetypes := kvinstancetypev1beta1.VirtualMachineInstancetypeList{}
	if err := client.List(ctx, &namespacedInstancetypes, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, err
	}
	for i := range namespacedInstancetypes.Items {
		if strings.EqualFold(namespacedInstancetypes.Items[i].Name, name) {
			return instanceTypeToNodeCapacity(namespacedInstancetypes.Items[i].Spec)
		}
	}
	return nil, nil
}

// instanceTypeToNodeCapacity extracts cpu, mem and gpu resource requests from the kubevirt instancetype.
func instanceTypeToNodeCapacity(it kvinstancetypev1beta1.VirtualMachineInstancetypeSpec) (*provider.NodeCapacity, error) {
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
