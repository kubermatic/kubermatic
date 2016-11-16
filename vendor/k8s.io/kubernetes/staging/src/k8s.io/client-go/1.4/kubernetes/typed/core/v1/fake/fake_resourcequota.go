/*
Copyright 2016 The Kubernetes Authors.

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

package fake

import (
	api "k8s.io/client-go/1.4/pkg/api"
	unversioned "k8s.io/client-go/1.4/pkg/api/unversioned"
	v1 "k8s.io/client-go/1.4/pkg/api/v1"
	labels "k8s.io/client-go/1.4/pkg/labels"
	watch "k8s.io/client-go/1.4/pkg/watch"
	testing "k8s.io/client-go/1.4/testing"
)

// FakeResourceQuotas implements ResourceQuotaInterface
type FakeResourceQuotas struct {
	Fake *FakeCore
	ns   string
}

var resourcequotasResource = unversioned.GroupVersionResource{Group: "", Version: "v1", Resource: "resourcequotas"}

func (c *FakeResourceQuotas) Create(resourceQuota *v1.ResourceQuota) (result *v1.ResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(resourcequotasResource, c.ns, resourceQuota), &v1.ResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ResourceQuota), err
}

func (c *FakeResourceQuotas) Update(resourceQuota *v1.ResourceQuota) (result *v1.ResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(resourcequotasResource, c.ns, resourceQuota), &v1.ResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ResourceQuota), err
}

func (c *FakeResourceQuotas) UpdateStatus(resourceQuota *v1.ResourceQuota) (*v1.ResourceQuota, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(resourcequotasResource, "status", c.ns, resourceQuota), &v1.ResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ResourceQuota), err
}

func (c *FakeResourceQuotas) Delete(name string, options *api.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(resourcequotasResource, c.ns, name), &v1.ResourceQuota{})

	return err
}

func (c *FakeResourceQuotas) DeleteCollection(options *api.DeleteOptions, listOptions api.ListOptions) error {
	action := testing.NewDeleteCollectionAction(resourcequotasResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1.ResourceQuotaList{})
	return err
}

func (c *FakeResourceQuotas) Get(name string) (result *v1.ResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(resourcequotasResource, c.ns, name), &v1.ResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ResourceQuota), err
}

func (c *FakeResourceQuotas) List(opts api.ListOptions) (result *v1.ResourceQuotaList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(resourcequotasResource, c.ns, opts), &v1.ResourceQuotaList{})

	if obj == nil {
		return nil, err
	}

	label := opts.LabelSelector
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.ResourceQuotaList{}
	for _, item := range obj.(*v1.ResourceQuotaList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested resourceQuotas.
func (c *FakeResourceQuotas) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(resourcequotasResource, c.ns, opts))

}

// Patch applies the patch and returns the patched resourceQuota.
func (c *FakeResourceQuotas) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.ResourceQuota, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(resourcequotasResource, c.ns, name, data, subresources...), &v1.ResourceQuota{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.ResourceQuota), err
}
