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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	VirtualMachineInstancetypeKind        = "VirtualMachineInstancetype"
	VirtualMachineClusterInstancetypeKind = "VirtualMachineClusterInstancetype"
)

// InstanceTypeSource identifies the KubeVirt instancetype used to resolve a VM's resources.
// It is diagnostic resolution metadata, not an immutable version or infrastructure reservation.
type InstanceTypeSource struct {
	Kind      string
	Namespace string
	Name      string
}

// ResolvedInstanceType contains the resources and source identity of a KubeVirt instancetype.
// DeviceResources preserves the exact device-plugin resource names requested by the
// instancetype.
type ResolvedInstanceType struct {
	Capacity        *provider.NodeCapacity
	DeviceResources corev1.ResourceList
	Source          InstanceTypeSource
}

type instanceTypeSpec struct {
	Spec   kvinstancetypev1beta1.VirtualMachineInstancetypeSpec
	Source InstanceTypeSource
}

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
	instanceType, err := findInstanceType(ctx, client, namespace, it)
	if err != nil {
		return nil, err
	}
	return instanceTypeToNodeCapacity(instanceType.Spec)
}

// ResolveInstanceType returns the capacity, exact device resource requests, and source
// identity from the selected VirtualMachine instancetype.
//
// [Alpha]: this function is part of the alpha KubeVirt accelerator quota integration.
//
// namespace scopes custom VirtualMachineInstancetype lookup to an infrastructure tenant.
func ResolveInstanceType(ctx context.Context, kubeconfig string, namespace string, it *kubevirtv1.InstancetypeMatcher) (*ResolvedInstanceType, error) {
	client, err := NewClient(kubeconfig, ClientOptions{})
	if err != nil {
		return nil, err
	}
	return resolveInstanceType(ctx, client, namespace, it)
}

func resolveInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace string, it *kubevirtv1.InstancetypeMatcher) (*ResolvedInstanceType, error) {
	if it == nil {
		return nil, fmt.Errorf("VMI instancetype matcher must not be nil")
	}

	instanceType, err := findInstanceTypeForResolution(ctx, client, namespace, it)
	if err != nil {
		return nil, err
	}
	return resolveInstanceTypeSpec(instanceType.Spec, instanceType.Source)
}

func findInstanceTypeForResolution(ctx context.Context, client ctrlruntimeclient.Client, namespace string, it *kubevirtv1.InstancetypeMatcher) (*instanceTypeSpec, error) {
	switch it.Kind {
	case VirtualMachineInstancetypeKind:
		if instanceType, err := findLiveNamespacedInstanceType(ctx, client, namespace, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}

	case VirtualMachineClusterInstancetypeKind:
		if instanceType, err := findClusterInstanceType(ctx, client, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}

	case "":
		// machine-controller resolves legacy matchers without a kind in namespace scope first.
		// Keep the accelerator footprint aligned with the instancetype used to provision the VM.
		if namespace != "" {
			if instanceType, err := findLiveNamespacedInstanceType(ctx, client, namespace, it.Name); instanceType != nil || err != nil {
				return instanceType, err
			}
		}
		if instanceType, err := findClusterInstanceType(ctx, client, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}
	}

	return nil, fmt.Errorf("VMI instancetype %s of Kind %s not found", it.Name, it.Kind)
}

func findInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace string, it *kubevirtv1.InstancetypeMatcher) (*instanceTypeSpec, error) {
	switch it.Kind {
	case VirtualMachineInstancetypeKind: // namespaced: kubermatic standard or user-deployed custom
		if instanceType, err := findNamespacedInstanceType(ctx, client, namespace, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}

	case VirtualMachineClusterInstancetypeKind: // cluster-wide
		if instanceType, err := findClusterInstanceType(ctx, client, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}

	case "": // kind absent — saved before kind was added to the API; search both types
		if instanceType, err := findClusterInstanceType(ctx, client, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}
		if instanceType, err := findNamespacedInstanceType(ctx, client, namespace, it.Name); instanceType != nil || err != nil {
			return instanceType, err
		}
	}
	return nil, fmt.Errorf("VMI instancetype %s of Kind %s not found", it.Name, it.Kind)
}

func findClusterInstanceType(ctx context.Context, client ctrlruntimeclient.Client, name string) (*instanceTypeSpec, error) {
	clusterInstancetypes := kvinstancetypev1beta1.VirtualMachineClusterInstancetypeList{}
	if err := client.List(ctx, &clusterInstancetypes); err != nil {
		return nil, err
	}
	for _, instancetype := range clusterInstancetypes.Items {
		if strings.EqualFold(instancetype.Name, name) {
			return &instanceTypeSpec{
				Spec: instancetype.Spec,
				Source: InstanceTypeSource{
					Kind: VirtualMachineClusterInstancetypeKind,
					Name: instancetype.Name,
				},
			}, nil
		}
	}
	return nil, nil
}

func findNamespacedInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) (*instanceTypeSpec, error) {
	standardInstancetypes := GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})
	for _, instancetype := range standardInstancetypes {
		if strings.EqualFold(instancetype.Name, name) {
			return &instanceTypeSpec{
				Spec: instancetype.Spec,
				Source: InstanceTypeSource{
					Kind:      VirtualMachineInstancetypeKind,
					Namespace: namespace,
					Name:      instancetype.Name,
				},
			}, nil
		}
	}
	return findLiveNamespacedInstanceType(ctx, client, namespace, name)
}

func findLiveNamespacedInstanceType(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) (*instanceTypeSpec, error) {
	// List namespaced VirtualMachineInstancetype objects from the infrastructure cluster.
	//
	// This requires the cluster's infra namespace: a namespaced instancetype must be resolved
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
			instancetype := &namespacedInstancetypes.Items[i]
			return &instanceTypeSpec{
				Spec: instancetype.Spec,
				Source: InstanceTypeSource{
					Kind:      VirtualMachineInstancetypeKind,
					Namespace: instancetype.Namespace,
					Name:      instancetype.Name,
				},
			}, nil
		}
	}
	return nil, nil
}

// instanceTypeToNodeCapacity extracts CPU and memory requests from the KubeVirt instancetype.
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

func resolveInstanceTypeSpec(it kvinstancetypev1beta1.VirtualMachineInstancetypeSpec, source InstanceTypeSource) (*ResolvedInstanceType, error) {
	capacity, err := instanceTypeToNodeCapacity(it)
	if err != nil {
		return nil, err
	}

	var deviceResources corev1.ResourceList
	for i, gpu := range it.GPUs {
		deviceResources, err = addDeviceResource(deviceResources, fmt.Sprintf("spec.gpus[%d].deviceName", i), gpu.DeviceName)
		if err != nil {
			return nil, err
		}
	}
	for i, hostDevice := range it.HostDevices {
		deviceResources, err = addDeviceResource(deviceResources, fmt.Sprintf("spec.hostDevices[%d].deviceName", i), hostDevice.DeviceName)
		if err != nil {
			return nil, err
		}
	}

	return &ResolvedInstanceType{
		Capacity:        capacity,
		DeviceResources: deviceResources,
		Source:          source,
	}, nil
}

func addDeviceResource(resources corev1.ResourceList, fieldPath, deviceName string) (corev1.ResourceList, error) {
	// The KubeVirt API version currently used by KKP requires deviceName for both GPUs and
	// host devices. Future DRA-backed entries need an explicit resolver rather than being
	// silently omitted from the quota footprint.
	if deviceName == "" {
		return nil, fmt.Errorf("%s must not be empty", fieldPath)
	}
	if strings.Count(deviceName, "/") != 1 {
		return nil, fmt.Errorf("%s %q must be a qualified resource name with exactly one '/'", fieldPath, deviceName)
	}
	if validationErrors := k8svalidation.IsQualifiedName(deviceName); len(validationErrors) > 0 {
		return nil, fmt.Errorf("%s %q is not a valid qualified resource name: %s", fieldPath, deviceName, strings.Join(validationErrors, "; "))
	}
	if resources == nil {
		resources = corev1.ResourceList{}
	}

	resourceName := corev1.ResourceName(deviceName)
	quantity := resources[resourceName]
	quantity.Add(*resource.NewQuantity(1, resource.DecimalSI))
	resources[resourceName] = quantity
	return resources, nil
}
