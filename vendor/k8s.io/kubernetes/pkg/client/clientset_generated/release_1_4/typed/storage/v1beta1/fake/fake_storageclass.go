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
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	v1beta1 "k8s.io/kubernetes/pkg/apis/storage/v1beta1"
	core "k8s.io/kubernetes/pkg/client/testing/core"
	labels "k8s.io/kubernetes/pkg/labels"
	watch "k8s.io/kubernetes/pkg/watch"
)

// FakeStorageClasses implements StorageClassInterface
type FakeStorageClasses struct {
	Fake *FakeStorage
}

var storageclassesResource = unversioned.GroupVersionResource{Group: "storage.k8s.io", Version: "v1beta1", Resource: "storageclasses"}

func (c *FakeStorageClasses) Create(storageClass *v1beta1.StorageClass) (result *v1beta1.StorageClass, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootCreateAction(storageclassesResource, storageClass), &v1beta1.StorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.StorageClass), err
}

func (c *FakeStorageClasses) Update(storageClass *v1beta1.StorageClass) (result *v1beta1.StorageClass, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootUpdateAction(storageclassesResource, storageClass), &v1beta1.StorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.StorageClass), err
}

func (c *FakeStorageClasses) Delete(name string, options *api.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewRootDeleteAction(storageclassesResource, name), &v1beta1.StorageClass{})
	return err
}

func (c *FakeStorageClasses) DeleteCollection(options *api.DeleteOptions, listOptions api.ListOptions) error {
	action := core.NewRootDeleteCollectionAction(storageclassesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.StorageClassList{})
	return err
}

func (c *FakeStorageClasses) Get(name string) (result *v1beta1.StorageClass, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootGetAction(storageclassesResource, name), &v1beta1.StorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.StorageClass), err
}

func (c *FakeStorageClasses) List(opts api.ListOptions) (result *v1beta1.StorageClassList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootListAction(storageclassesResource, opts), &v1beta1.StorageClassList{})
	if obj == nil {
		return nil, err
	}

	label := opts.LabelSelector
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.StorageClassList{}
	for _, item := range obj.(*v1beta1.StorageClassList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested storageClasses.
func (c *FakeStorageClasses) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewRootWatchAction(storageclassesResource, opts))
}

// Patch applies the patch and returns the patched storageClass.
func (c *FakeStorageClasses) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1beta1.StorageClass, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootPatchSubresourceAction(storageclassesResource, name, data, subresources...), &v1beta1.StorageClass{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.StorageClass), err
}
