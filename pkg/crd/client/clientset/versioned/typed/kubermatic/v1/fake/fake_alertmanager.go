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

// FakeAlertmanagers implements AlertmanagerInterface
type FakeAlertmanagers struct {
	Fake *FakeKubermaticV1
	ns   string
}

var alertmanagersResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "alertmanagers"}

var alertmanagersKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "Alertmanager"}

// Get takes name of the alertmanager, and returns the corresponding alertmanager object, and an error if there is any.
func (c *FakeAlertmanagers) Get(ctx context.Context, name string, options v1.GetOptions) (result *kubermaticv1.Alertmanager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(alertmanagersResource, c.ns, name), &kubermaticv1.Alertmanager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.Alertmanager), err
}

// List takes label and field selectors, and returns the list of Alertmanagers that match those selectors.
func (c *FakeAlertmanagers) List(ctx context.Context, opts v1.ListOptions) (result *kubermaticv1.AlertmanagerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(alertmanagersResource, alertmanagersKind, c.ns, opts), &kubermaticv1.AlertmanagerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermaticv1.AlertmanagerList{ListMeta: obj.(*kubermaticv1.AlertmanagerList).ListMeta}
	for _, item := range obj.(*kubermaticv1.AlertmanagerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested alertmanagers.
func (c *FakeAlertmanagers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(alertmanagersResource, c.ns, opts))

}

// Create takes the representation of a alertmanager and creates it.  Returns the server's representation of the alertmanager, and an error, if there is any.
func (c *FakeAlertmanagers) Create(ctx context.Context, alertmanager *kubermaticv1.Alertmanager, opts v1.CreateOptions) (result *kubermaticv1.Alertmanager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(alertmanagersResource, c.ns, alertmanager), &kubermaticv1.Alertmanager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.Alertmanager), err
}

// Update takes the representation of a alertmanager and updates it. Returns the server's representation of the alertmanager, and an error, if there is any.
func (c *FakeAlertmanagers) Update(ctx context.Context, alertmanager *kubermaticv1.Alertmanager, opts v1.UpdateOptions) (result *kubermaticv1.Alertmanager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(alertmanagersResource, c.ns, alertmanager), &kubermaticv1.Alertmanager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.Alertmanager), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAlertmanagers) UpdateStatus(ctx context.Context, alertmanager *kubermaticv1.Alertmanager, opts v1.UpdateOptions) (*kubermaticv1.Alertmanager, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(alertmanagersResource, "status", c.ns, alertmanager), &kubermaticv1.Alertmanager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.Alertmanager), err
}

// Delete takes name of the alertmanager and deletes it. Returns an error if one occurs.
func (c *FakeAlertmanagers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(alertmanagersResource, c.ns, name, opts), &kubermaticv1.Alertmanager{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAlertmanagers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(alertmanagersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &kubermaticv1.AlertmanagerList{})
	return err
}

// Patch applies the patch and returns the patched alertmanager.
func (c *FakeAlertmanagers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *kubermaticv1.Alertmanager, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(alertmanagersResource, c.ns, name, pt, data, subresources...), &kubermaticv1.Alertmanager{})

	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.Alertmanager), err
}
