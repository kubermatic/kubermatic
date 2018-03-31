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

// FakeClusters implements ClusterInterface
type FakeClusters struct {
	Fake *FakeKubermaticV1
}

var clustersResource = schema.GroupVersionResource{Group: "kubermatic.k8s.io", Version: "v1", Resource: "clusters"}

var clustersKind = schema.GroupVersionKind{Group: "kubermatic.k8s.io", Version: "v1", Kind: "Cluster"}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *FakeClusters) Get(name string, options v1.GetOptions) (result *kubermatic_v1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clustersResource, name), &kubermatic_v1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.Cluster), err
}

// List takes label and field selectors, and returns the list of Clusters that match those selectors.
func (c *FakeClusters) List(opts v1.ListOptions) (result *kubermatic_v1.ClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clustersResource, clustersKind, opts), &kubermatic_v1.ClusterList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubermatic_v1.ClusterList{}
	for _, item := range obj.(*kubermatic_v1.ClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *FakeClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clustersResource, opts))
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Create(cluster *kubermatic_v1.Cluster) (result *kubermatic_v1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clustersResource, cluster), &kubermatic_v1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.Cluster), err
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *FakeClusters) Update(cluster *kubermatic_v1.Cluster) (result *kubermatic_v1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clustersResource, cluster), &kubermatic_v1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.Cluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusters) UpdateStatus(cluster *kubermatic_v1.Cluster) (*kubermatic_v1.Cluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(clustersResource, "status", cluster), &kubermatic_v1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.Cluster), err
}

// Delete takes name of the cluster and deletes it. Returns an error if one occurs.
func (c *FakeClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clustersResource, name), &kubermatic_v1.Cluster{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clustersResource, listOptions)

	_, err := c.Fake.Invokes(action, &kubermatic_v1.ClusterList{})
	return err
}

// Patch applies the patch and returns the patched cluster.
func (c *FakeClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *kubermatic_v1.Cluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clustersResource, name, data, subresources...), &kubermatic_v1.Cluster{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubermatic_v1.Cluster), err
}
