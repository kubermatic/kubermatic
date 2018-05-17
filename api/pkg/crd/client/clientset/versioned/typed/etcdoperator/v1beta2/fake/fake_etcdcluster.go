package fake

import (
	v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEtcdClusters implements EtcdClusterInterface
type FakeEtcdClusters struct {
	Fake *FakeEtcdV1beta2
	ns   string
}

var etcdclustersResource = schema.GroupVersionResource{Group: "etcd.database.coreos.com", Version: "v1beta2", Resource: "etcdclusters"}

var etcdclustersKind = schema.GroupVersionKind{Group: "etcd.database.coreos.com", Version: "v1beta2", Kind: "EtcdCluster"}

// Get takes name of the etcdCluster, and returns the corresponding etcdCluster object, and an error if there is any.
func (c *FakeEtcdClusters) Get(name string, options v1.GetOptions) (result *v1beta2.EtcdCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(etcdclustersResource, c.ns, name), &v1beta2.EtcdCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.EtcdCluster), err
}

// List takes label and field selectors, and returns the list of EtcdClusters that match those selectors.
func (c *FakeEtcdClusters) List(opts v1.ListOptions) (result *v1beta2.EtcdClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(etcdclustersResource, etcdclustersKind, c.ns, opts), &v1beta2.EtcdClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta2.EtcdClusterList{}
	for _, item := range obj.(*v1beta2.EtcdClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested etcdClusters.
func (c *FakeEtcdClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(etcdclustersResource, c.ns, opts))

}

// Create takes the representation of a etcdCluster and creates it.  Returns the server's representation of the etcdCluster, and an error, if there is any.
func (c *FakeEtcdClusters) Create(etcdCluster *v1beta2.EtcdCluster) (result *v1beta2.EtcdCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(etcdclustersResource, c.ns, etcdCluster), &v1beta2.EtcdCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.EtcdCluster), err
}

// Update takes the representation of a etcdCluster and updates it. Returns the server's representation of the etcdCluster, and an error, if there is any.
func (c *FakeEtcdClusters) Update(etcdCluster *v1beta2.EtcdCluster) (result *v1beta2.EtcdCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(etcdclustersResource, c.ns, etcdCluster), &v1beta2.EtcdCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.EtcdCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeEtcdClusters) UpdateStatus(etcdCluster *v1beta2.EtcdCluster) (*v1beta2.EtcdCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(etcdclustersResource, "status", c.ns, etcdCluster), &v1beta2.EtcdCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.EtcdCluster), err
}

// Delete takes name of the etcdCluster and deletes it. Returns an error if one occurs.
func (c *FakeEtcdClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(etcdclustersResource, c.ns, name), &v1beta2.EtcdCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEtcdClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(etcdclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta2.EtcdClusterList{})
	return err
}

// Patch applies the patch and returns the patched etcdCluster.
func (c *FakeEtcdClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta2.EtcdCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(etcdclustersResource, c.ns, name, data, subresources...), &v1beta2.EtcdCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.EtcdCluster), err
}
