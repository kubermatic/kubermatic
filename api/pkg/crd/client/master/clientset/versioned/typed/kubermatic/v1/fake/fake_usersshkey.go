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

// FakeUserSSHKeies implements UserSSHKeyInterface
type FakeUserSSHKeies struct {
	Fake *FakeKubermaticV1
}

var usersshkeiesResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "usersshkeies"}

var usersshkeiesKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "UserSSHKey"}

// Get takes name of the userSSHKey, and returns the corresponding userSSHKey object, and an error if there is any.
func (c *FakeUserSSHKeies) Get(name string, options v1.GetOptions) (result *kubermatic_v1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(usersshkeiesResource, name), &kubermatic_v1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserSSHKey), err
}

// List takes label and field selectors, and returns the list of UserSSHKeies that match those selectors.
func (c *FakeUserSSHKeies) List(opts v1.ListOptions) (result *kubermatic_v1.UserSSHKeyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(usersshkeiesResource, usersshkeiesKind, opts), &kubermatic_v1.UserSSHKeyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermatic_v1.UserSSHKeyList{}
	for _, item := range obj.(*kubermatic_v1.UserSSHKeyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested userSSHKeies.
func (c *FakeUserSSHKeies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(usersshkeiesResource, opts))
}

// Create takes the representation of a userSSHKey and creates it.  Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *FakeUserSSHKeies) Create(userSSHKey *kubermatic_v1.UserSSHKey) (result *kubermatic_v1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(usersshkeiesResource, userSSHKey), &kubermatic_v1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserSSHKey), err
}

// Update takes the representation of a userSSHKey and updates it. Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *FakeUserSSHKeies) Update(userSSHKey *kubermatic_v1.UserSSHKey) (result *kubermatic_v1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(usersshkeiesResource, userSSHKey), &kubermatic_v1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserSSHKey), err
}

// Delete takes name of the userSSHKey and deletes it. Returns an error if one occurs.
func (c *FakeUserSSHKeies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(usersshkeiesResource, name), &kubermatic_v1.UserSSHKey{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUserSSHKeies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(usersshkeiesResource, listOptions)

	_, err := c.Fake.Invokes(action, &kubermatic_v1.UserSSHKeyList{})
	return err
}

// Patch applies the patch and returns the patched userSSHKey.
func (c *FakeUserSSHKeies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubermatic_v1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(usersshkeiesResource, name, data, subresources...), &kubermatic_v1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.UserSSHKey), err
}
