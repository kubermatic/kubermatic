package fake

import (
	kubermatic_v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeUserProjectBindings implements UserProjectBindingInterface
type FakeUserProjectBindings struct {
	Fake *FakeKubermaticV1
}

var userprojectbindingsResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "userprojectbindings"}

var userprojectbindingsKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "UserProjectBinding"}

// Get takes name of the userProjectBinding, and returns the corresponding userProjectBinding object, and an error if there is any.
func (c *FakeUserProjectBindings) Get(name string, options v1.GetOptions) (result *kubermatic_v1.UserProjectBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(userprojectbindingsResource, name), &kubermatic_v1.UserProjectBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserProjectBinding), err
}

// List takes label and field selectors, and returns the list of UserProjectBindings that match those selectors.
func (c *FakeUserProjectBindings) List(opts v1.ListOptions) (result *kubermatic_v1.UserProjectBindingList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(userprojectbindingsResource, userprojectbindingsKind, opts), &kubermatic_v1.UserProjectBindingList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermatic_v1.UserProjectBindingList{}
	for _, item := range obj.(*kubermatic_v1.UserProjectBindingList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested userProjectBindings.
func (c *FakeUserProjectBindings) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(userprojectbindingsResource, opts))
}

// Create takes the representation of a userProjectBinding and creates it.  Returns the server's representation of the userProjectBinding, and an error, if there is any.
func (c *FakeUserProjectBindings) Create(userProjectBinding *kubermatic_v1.UserProjectBinding) (result *kubermatic_v1.UserProjectBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(userprojectbindingsResource, userProjectBinding), &kubermatic_v1.UserProjectBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserProjectBinding), err
}

// Update takes the representation of a userProjectBinding and updates it. Returns the server's representation of the userProjectBinding, and an error, if there is any.
func (c *FakeUserProjectBindings) Update(userProjectBinding *kubermatic_v1.UserProjectBinding) (result *kubermatic_v1.UserProjectBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(userprojectbindingsResource, userProjectBinding), &kubermatic_v1.UserProjectBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserProjectBinding), err
}

// Delete takes name of the userProjectBinding and deletes it. Returns an error if one occurs.
func (c *FakeUserProjectBindings) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(userprojectbindingsResource, name), &kubermatic_v1.UserProjectBinding{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUserProjectBindings) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(userprojectbindingsResource, listOptions)

	_, err := c.Fake.Invokes(action, &kubermatic_v1.UserProjectBindingList{})
	return err
}

// Patch applies the patch and returns the patched userProjectBinding.
func (c *FakeUserProjectBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubermatic_v1.UserProjectBinding, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(userprojectbindingsResource, name, data, subresources...), &kubermatic_v1.UserProjectBinding{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserProjectBinding), err
}
