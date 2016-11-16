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
	v1alpha1 "k8s.io/client-go/1.4/pkg/apis/rbac/v1alpha1"
	labels "k8s.io/client-go/1.4/pkg/labels"
	watch "k8s.io/client-go/1.4/pkg/watch"
	testing "k8s.io/client-go/1.4/testing"
)

// FakeClusterRoleBindings implements ClusterRoleBindingInterface
type FakeClusterRoleBindings struct {
	Fake *FakeRbac
}

var clusterrolebindingsResource = unversioned.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1alpha1", Resource: "clusterrolebindings"}

func (c *FakeClusterRoleBindings) Create(clusterRoleBinding *v1alpha1.ClusterRoleBinding) (result *v1alpha1.ClusterRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterrolebindingsResource, clusterRoleBinding), &v1alpha1.ClusterRoleBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterRoleBinding), err
}

func (c *FakeClusterRoleBindings) Update(clusterRoleBinding *v1alpha1.ClusterRoleBinding) (result *v1alpha1.ClusterRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterrolebindingsResource, clusterRoleBinding), &v1alpha1.ClusterRoleBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterRoleBinding), err
}

func (c *FakeClusterRoleBindings) Delete(name string, options *api.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusterrolebindingsResource, name), &v1alpha1.ClusterRoleBinding{})
	return err
}

func (c *FakeClusterRoleBindings) DeleteCollection(options *api.DeleteOptions, listOptions api.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterrolebindingsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterRoleBindingList{})
	return err
}

func (c *FakeClusterRoleBindings) Get(name string) (result *v1alpha1.ClusterRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterrolebindingsResource, name), &v1alpha1.ClusterRoleBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterRoleBinding), err
}

func (c *FakeClusterRoleBindings) List(opts api.ListOptions) (result *v1alpha1.ClusterRoleBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterrolebindingsResource, opts), &v1alpha1.ClusterRoleBindingList{})
	if obj == nil {
		return nil, err
	}

	label := opts.LabelSelector
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterRoleBindingList{}
	for _, item := range obj.(*v1alpha1.ClusterRoleBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterRoleBindings.
func (c *FakeClusterRoleBindings) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterrolebindingsResource, opts))
}

// Patch applies the patch and returns the patched clusterRoleBinding.
func (c *FakeClusterRoleBindings) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterRoleBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterrolebindingsResource, name, data, subresources...), &v1alpha1.ClusterRoleBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterRoleBinding), err
}
