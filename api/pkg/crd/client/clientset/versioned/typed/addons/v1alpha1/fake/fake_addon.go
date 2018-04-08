package fake

import (
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/addons/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAddons implements AddonInterface
type FakeAddons struct {
	Fake *FakeAddonsV1alpha1
}

var addonsResource = schema.GroupVersionResource{Group: "addons.k8s.io", Version: "v1alpha1", Resource: "addons"}

var addonsKind = schema.GroupVersionKind{Group: "addons.k8s.io", Version: "v1alpha1", Kind: "Addon"}

// Get takes name of the addon, and returns the corresponding addon object, and an error if there is any.
func (c *FakeAddons) Get(name string, options v1.GetOptions) (result *v1alpha1.Addon, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(addonsResource, name), &v1alpha1.Addon{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Addon), err
}

// List takes label and field selectors, and returns the list of Addons that match those selectors.
func (c *FakeAddons) List(opts v1.ListOptions) (result *v1alpha1.AddonList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(addonsResource, addonsKind, opts), &v1alpha1.AddonList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AddonList{}
	for _, item := range obj.(*v1alpha1.AddonList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested addons.
func (c *FakeAddons) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(addonsResource, opts))
}

// Create takes the representation of a addon and creates it.  Returns the server's representation of the addon, and an error, if there is any.
func (c *FakeAddons) Create(addon *v1alpha1.Addon) (result *v1alpha1.Addon, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(addonsResource, addon), &v1alpha1.Addon{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Addon), err
}

// Update takes the representation of a addon and updates it. Returns the server's representation of the addon, and an error, if there is any.
func (c *FakeAddons) Update(addon *v1alpha1.Addon) (result *v1alpha1.Addon, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(addonsResource, addon), &v1alpha1.Addon{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Addon), err
}

// Delete takes name of the addon and deletes it. Returns an error if one occurs.
func (c *FakeAddons) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(addonsResource, name), &v1alpha1.Addon{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAddons) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(addonsResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.AddonList{})
	return err
}

// Patch applies the patch and returns the patched addon.
func (c *FakeAddons) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Addon, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(addonsResource, name, data, subresources...), &v1alpha1.Addon{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Addon), err
}
