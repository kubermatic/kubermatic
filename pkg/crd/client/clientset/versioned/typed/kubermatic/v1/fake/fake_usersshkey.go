// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeUserSSHKeys implements UserSSHKeyInterface
type FakeUserSSHKeys struct {
	Fake *FakeKubermaticV1
}

var usersshkeysResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "usersshkeies"}

var usersshkeysKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "UserSSHKey"}

// Get takes name of the userSSHKey, and returns the corresponding userSSHKey object, and an error if there is any.
func (c *FakeUserSSHKeys) Get(ctx context.Context, name string, options v1.GetOptions) (result *kubermaticv1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(usersshkeysResource, name), &kubermaticv1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.UserSSHKey), err
}

// List takes label and field selectors, and returns the list of UserSSHKeys that match those selectors.
func (c *FakeUserSSHKeys) List(ctx context.Context, opts v1.ListOptions) (result *kubermaticv1.UserSSHKeyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(usersshkeysResource, usersshkeysKind, opts), &kubermaticv1.UserSSHKeyList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermaticv1.UserSSHKeyList{ListMeta: obj.(*kubermaticv1.UserSSHKeyList).ListMeta}
	for _, item := range obj.(*kubermaticv1.UserSSHKeyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested userSSHKeys.
func (c *FakeUserSSHKeys) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(usersshkeysResource, opts))
}

// Create takes the representation of a userSSHKey and creates it.  Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *FakeUserSSHKeys) Create(ctx context.Context, userSSHKey *kubermaticv1.UserSSHKey, opts v1.CreateOptions) (result *kubermaticv1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(usersshkeysResource, userSSHKey), &kubermaticv1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.UserSSHKey), err
}

// Update takes the representation of a userSSHKey and updates it. Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *FakeUserSSHKeys) Update(ctx context.Context, userSSHKey *kubermaticv1.UserSSHKey, opts v1.UpdateOptions) (result *kubermaticv1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(usersshkeysResource, userSSHKey), &kubermaticv1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.UserSSHKey), err
}

// Delete takes name of the userSSHKey and deletes it. Returns an error if one occurs.
func (c *FakeUserSSHKeys) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(usersshkeysResource, name), &kubermaticv1.UserSSHKey{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeUserSSHKeys) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(usersshkeysResource, listOpts)

	_, err := c.Fake.Invokes(action, &kubermaticv1.UserSSHKeyList{})
	return err
}

// Patch applies the patch and returns the patched userSSHKey.
func (c *FakeUserSSHKeys) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *kubermaticv1.UserSSHKey, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(usersshkeysResource, name, pt, data, subresources...), &kubermaticv1.UserSSHKey{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.UserSSHKey), err
}
