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

// FakeAllowedRegistries implements AllowedRegistryInterface
type FakeAllowedRegistries struct {
	Fake *FakeKubermaticV1
}

var allowedregistriesResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "allowedregistries"}

var allowedregistriesKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "AllowedRegistry"}

// Get takes name of the allowedRegistry, and returns the corresponding allowedRegistry object, and an error if there is any.
func (c *FakeAllowedRegistries) Get(ctx context.Context, name string, options v1.GetOptions) (result *kubermaticv1.AllowedRegistry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(allowedregistriesResource, name), &kubermaticv1.AllowedRegistry{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.AllowedRegistry), err
}

// List takes label and field selectors, and returns the list of AllowedRegistries that match those selectors.
func (c *FakeAllowedRegistries) List(ctx context.Context, opts v1.ListOptions) (result *kubermaticv1.AllowedRegistryList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(allowedregistriesResource, allowedregistriesKind, opts), &kubermaticv1.AllowedRegistryList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermaticv1.AllowedRegistryList{ListMeta: obj.(*kubermaticv1.AllowedRegistryList).ListMeta}
	for _, item := range obj.(*kubermaticv1.AllowedRegistryList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested allowedRegistries.
func (c *FakeAllowedRegistries) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(allowedregistriesResource, opts))
}

// Create takes the representation of a allowedRegistry and creates it.  Returns the server's representation of the allowedRegistry, and an error, if there is any.
func (c *FakeAllowedRegistries) Create(ctx context.Context, allowedRegistry *kubermaticv1.AllowedRegistry, opts v1.CreateOptions) (result *kubermaticv1.AllowedRegistry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(allowedregistriesResource, allowedRegistry), &kubermaticv1.AllowedRegistry{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.AllowedRegistry), err
}

// Update takes the representation of a allowedRegistry and updates it. Returns the server's representation of the allowedRegistry, and an error, if there is any.
func (c *FakeAllowedRegistries) Update(ctx context.Context, allowedRegistry *kubermaticv1.AllowedRegistry, opts v1.UpdateOptions) (result *kubermaticv1.AllowedRegistry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(allowedregistriesResource, allowedRegistry), &kubermaticv1.AllowedRegistry{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.AllowedRegistry), err
}

// Delete takes name of the allowedRegistry and deletes it. Returns an error if one occurs.
func (c *FakeAllowedRegistries) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(allowedregistriesResource, name), &kubermaticv1.AllowedRegistry{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAllowedRegistries) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(allowedregistriesResource, listOpts)

	_, err := c.Fake.Invokes(action, &kubermaticv1.AllowedRegistryList{})
	return err
}

// Patch applies the patch and returns the patched allowedRegistry.
func (c *FakeAllowedRegistries) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *kubermaticv1.AllowedRegistry, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(allowedregistriesResource, name, pt, data, subresources...), &kubermaticv1.AllowedRegistry{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.AllowedRegistry), err
}
