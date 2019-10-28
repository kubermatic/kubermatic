/*
Copyright The Kubernetes Authors.

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
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeLeases implements LeaseInterface
type FakeLeases struct {
	Fake *FakeCoordinationV1
	ns   string
}

var leasesResource = schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}

var leasesKind = schema.GroupVersionKind{Group: "coordination.k8s.io", Version: "v1", Kind: "Lease"}

// Get takes name of the lease, and returns the corresponding lease object, and an error if there is any.
func (c *FakeLeases) Get(name string, options v1.GetOptions) (result *coordinationv1.Lease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(leasesResource, c.ns, name), &coordinationv1.Lease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), err
}

// List takes label and field selectors, and returns the list of Leases that match those selectors.
func (c *FakeLeases) List(opts v1.ListOptions) (result *coordinationv1.LeaseList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(leasesResource, leasesKind, c.ns, opts), &coordinationv1.LeaseList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &coordinationv1.LeaseList{ListMeta: obj.(*coordinationv1.LeaseList).ListMeta}
	for _, item := range obj.(*coordinationv1.LeaseList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested leases.
func (c *FakeLeases) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(leasesResource, c.ns, opts))

}

// Create takes the representation of a lease and creates it.  Returns the server's representation of the lease, and an error, if there is any.
func (c *FakeLeases) Create(lease *coordinationv1.Lease) (result *coordinationv1.Lease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(leasesResource, c.ns, lease), &coordinationv1.Lease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), err
}

// Update takes the representation of a lease and updates it. Returns the server's representation of the lease, and an error, if there is any.
func (c *FakeLeases) Update(lease *coordinationv1.Lease) (result *coordinationv1.Lease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(leasesResource, c.ns, lease), &coordinationv1.Lease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), err
}

// Delete takes name of the lease and deletes it. Returns an error if one occurs.
func (c *FakeLeases) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(leasesResource, c.ns, name), &coordinationv1.Lease{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeLeases) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(leasesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &coordinationv1.LeaseList{})
	return err
}

// Patch applies the patch and returns the patched lease.
func (c *FakeLeases) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *coordinationv1.Lease, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(leasesResource, c.ns, name, pt, data, subresources...), &coordinationv1.Lease{})

	if obj == nil {
		return nil, err
	}
	return obj.(*coordinationv1.Lease), err
}
