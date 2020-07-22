// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeExternalClusters implements ExternalClusterInterface
type FakeExternalClusters struct {
	Fake *FakeKubermaticV1
}

var externalclustersResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "externalclusters"}

var externalclustersKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "ExternalCluster"}

// Get takes name of the externalCluster, and returns the corresponding externalCluster object, and an error if there is any.
func (c *FakeExternalClusters) Get(name string, options v1.GetOptions) (result *kubermaticv1.ExternalCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(externalclustersResource, name), &kubermaticv1.ExternalCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ExternalCluster), err
}

// List takes label and field selectors, and returns the list of ExternalClusters that match those selectors.
func (c *FakeExternalClusters) List(opts v1.ListOptions) (result *kubermaticv1.ExternalClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(externalclustersResource, externalclustersKind, opts), &kubermaticv1.ExternalClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermaticv1.ExternalClusterList{ListMeta: obj.(*kubermaticv1.ExternalClusterList).ListMeta}
	for _, item := range obj.(*kubermaticv1.ExternalClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested externalClusters.
func (c *FakeExternalClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(externalclustersResource, opts))
}

// Create takes the representation of a externalCluster and creates it.  Returns the server's representation of the externalCluster, and an error, if there is any.
func (c *FakeExternalClusters) Create(externalCluster *kubermaticv1.ExternalCluster) (result *kubermaticv1.ExternalCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(externalclustersResource, externalCluster), &kubermaticv1.ExternalCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ExternalCluster), err
}

// Update takes the representation of a externalCluster and updates it. Returns the server's representation of the externalCluster, and an error, if there is any.
func (c *FakeExternalClusters) Update(externalCluster *kubermaticv1.ExternalCluster) (result *kubermaticv1.ExternalCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(externalclustersResource, externalCluster), &kubermaticv1.ExternalCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ExternalCluster), err
}

// Delete takes name of the externalCluster and deletes it. Returns an error if one occurs.
func (c *FakeExternalClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(externalclustersResource, name), &kubermaticv1.ExternalCluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeExternalClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(externalclustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &kubermaticv1.ExternalClusterList{})
	return err
}

// Patch applies the patch and returns the patched externalCluster.
func (c *FakeExternalClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubermaticv1.ExternalCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(externalclustersResource, name, pt, data, subresources...), &kubermaticv1.ExternalCluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermaticv1.ExternalCluster), err
}
