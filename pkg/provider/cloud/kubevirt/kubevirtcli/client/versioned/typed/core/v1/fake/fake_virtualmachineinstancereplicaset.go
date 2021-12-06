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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	corev1 "kubevirt.io/api/core/v1"
)

// FakeVirtualMachineInstanceReplicaSets implements VirtualMachineInstanceReplicaSetInterface
type FakeVirtualMachineInstanceReplicaSets struct {
	Fake *FakeKubevirtV1
	ns   string
}

var virtualmachineinstancereplicasetsResource = schema.GroupVersionResource{Group: "kubevirt.io", Version: "v1", Resource: "virtualmachineinstancereplicasets"}

var virtualmachineinstancereplicasetsKind = schema.GroupVersionKind{Group: "kubevirt.io", Version: "v1", Kind: "VirtualMachineInstanceReplicaSet"}

// Get takes name of the virtualMachineInstanceReplicaSet, and returns the corresponding virtualMachineInstanceReplicaSet object, and an error if there is any.
func (c *FakeVirtualMachineInstanceReplicaSets) Get(ctx context.Context, name string, options v1.GetOptions) (result *corev1.VirtualMachineInstanceReplicaSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(virtualmachineinstancereplicasetsResource, c.ns, name), &corev1.VirtualMachineInstanceReplicaSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*corev1.VirtualMachineInstanceReplicaSet), err
}

// List takes label and field selectors, and returns the list of VirtualMachineInstanceReplicaSets that match those selectors.
func (c *FakeVirtualMachineInstanceReplicaSets) List(ctx context.Context, opts v1.ListOptions) (result *corev1.VirtualMachineInstanceReplicaSetList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(virtualmachineinstancereplicasetsResource, virtualmachineinstancereplicasetsKind, c.ns, opts), &corev1.VirtualMachineInstanceReplicaSetList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &corev1.VirtualMachineInstanceReplicaSetList{ListMeta: obj.(*corev1.VirtualMachineInstanceReplicaSetList).ListMeta}
	for _, item := range obj.(*corev1.VirtualMachineInstanceReplicaSetList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested virtualMachineInstanceReplicaSets.
func (c *FakeVirtualMachineInstanceReplicaSets) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(virtualmachineinstancereplicasetsResource, c.ns, opts))

}

// Create takes the representation of a virtualMachineInstanceReplicaSet and creates it.  Returns the server's representation of the virtualMachineInstanceReplicaSet, and an error, if there is any.
func (c *FakeVirtualMachineInstanceReplicaSets) Create(ctx context.Context, virtualMachineInstanceReplicaSet *corev1.VirtualMachineInstanceReplicaSet, opts v1.CreateOptions) (result *corev1.VirtualMachineInstanceReplicaSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(virtualmachineinstancereplicasetsResource, c.ns, virtualMachineInstanceReplicaSet), &corev1.VirtualMachineInstanceReplicaSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*corev1.VirtualMachineInstanceReplicaSet), err
}

// Update takes the representation of a virtualMachineInstanceReplicaSet and updates it. Returns the server's representation of the virtualMachineInstanceReplicaSet, and an error, if there is any.
func (c *FakeVirtualMachineInstanceReplicaSets) Update(ctx context.Context, virtualMachineInstanceReplicaSet *corev1.VirtualMachineInstanceReplicaSet, opts v1.UpdateOptions) (result *corev1.VirtualMachineInstanceReplicaSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(virtualmachineinstancereplicasetsResource, c.ns, virtualMachineInstanceReplicaSet), &corev1.VirtualMachineInstanceReplicaSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*corev1.VirtualMachineInstanceReplicaSet), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVirtualMachineInstanceReplicaSets) UpdateStatus(ctx context.Context, virtualMachineInstanceReplicaSet *corev1.VirtualMachineInstanceReplicaSet, opts v1.UpdateOptions) (*corev1.VirtualMachineInstanceReplicaSet, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(virtualmachineinstancereplicasetsResource, "status", c.ns, virtualMachineInstanceReplicaSet), &corev1.VirtualMachineInstanceReplicaSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*corev1.VirtualMachineInstanceReplicaSet), err
}

// Delete takes name of the virtualMachineInstanceReplicaSet and deletes it. Returns an error if one occurs.
func (c *FakeVirtualMachineInstanceReplicaSets) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(virtualmachineinstancereplicasetsResource, c.ns, name), &corev1.VirtualMachineInstanceReplicaSet{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVirtualMachineInstanceReplicaSets) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(virtualmachineinstancereplicasetsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &corev1.VirtualMachineInstanceReplicaSetList{})
	return err
}

// Patch applies the patch and returns the patched virtualMachineInstanceReplicaSet.
func (c *FakeVirtualMachineInstanceReplicaSets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *corev1.VirtualMachineInstanceReplicaSet, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(virtualmachineinstancereplicasetsResource, c.ns, name, pt, data, subresources...), &corev1.VirtualMachineInstanceReplicaSet{})

	if obj == nil {
		return nil, err
	}
	return obj.(*corev1.VirtualMachineInstanceReplicaSet), err
}
