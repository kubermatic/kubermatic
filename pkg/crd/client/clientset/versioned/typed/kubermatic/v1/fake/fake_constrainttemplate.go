// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeConstraintTemplates implements ConstraintTemplateInterface
type FakeConstraintTemplates struct {
	Fake *FakeKubermaticV1
}

var constrainttemplatesResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "constrainttemplates"}

var constrainttemplatesKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "ConstraintTemplate"}

// Get takes name of the constraintTemplate, and returns the corresponding constraintTemplate object, and an error if there is any.
func (c *FakeConstraintTemplates) Get(ctx context.Context, name string, options v1.GetOptions) (result *kubermaticv1.ConstraintTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(constrainttemplatesResource, name), &kubermaticv1.ConstraintTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ConstraintTemplate), err
}

// List takes label and field selectors, and returns the list of ConstraintTemplates that match those selectors.
func (c *FakeConstraintTemplates) List(ctx context.Context, opts v1.ListOptions) (result *kubermaticv1.ConstraintTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(constrainttemplatesResource, constrainttemplatesKind, opts), &kubermaticv1.ConstraintTemplateList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermaticv1.ConstraintTemplateList{ListMeta: obj.(*kubermaticv1.ConstraintTemplateList).ListMeta}
	for _, item := range obj.(*kubermaticv1.ConstraintTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested constraintTemplates.
func (c *FakeConstraintTemplates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(constrainttemplatesResource, opts))
}

// Create takes the representation of a constraintTemplate and creates it.  Returns the server's representation of the constraintTemplate, and an error, if there is any.
func (c *FakeConstraintTemplates) Create(ctx context.Context, constraintTemplate *kubermaticv1.ConstraintTemplate, opts v1.CreateOptions) (result *kubermaticv1.ConstraintTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(constrainttemplatesResource, constraintTemplate), &kubermaticv1.ConstraintTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ConstraintTemplate), err
}

// Update takes the representation of a constraintTemplate and updates it. Returns the server's representation of the constraintTemplate, and an error, if there is any.
func (c *FakeConstraintTemplates) Update(ctx context.Context, constraintTemplate *kubermaticv1.ConstraintTemplate, opts v1.UpdateOptions) (result *kubermaticv1.ConstraintTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(constrainttemplatesResource, constraintTemplate), &kubermaticv1.ConstraintTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ConstraintTemplate), err
}

// Delete takes name of the constraintTemplate and deletes it. Returns an error if one occurs.
func (c *FakeConstraintTemplates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(constrainttemplatesResource, name, opts), &kubermaticv1.ConstraintTemplate{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeConstraintTemplates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(constrainttemplatesResource, listOpts)

	_, err := c.Fake.Invokes(action, &kubermaticv1.ConstraintTemplateList{})
	return err
}

// Patch applies the patch and returns the patched constraintTemplate.
func (c *FakeConstraintTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *kubermaticv1.ConstraintTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(constrainttemplatesResource, name, pt, data, subresources...), &kubermaticv1.ConstraintTemplate{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ConstraintTemplate), err
}
