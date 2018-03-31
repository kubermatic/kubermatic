package fake

import (
	prometheus_v1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServiceMonitors implements ServiceMonitorInterface
type FakeServiceMonitors struct {
	Fake *FakeMonitoringV1
	ns   string
}

var servicemonitorsResource = schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors"}

var servicemonitorsKind = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "ServiceMonitor"}

// Get takes name of the serviceMonitor, and returns the corresponding serviceMonitor object, and an error if there is any.
func (c *FakeServiceMonitors) Get(name string, options v1.GetOptions) (result *prometheus_v1.ServiceMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(servicemonitorsResource, c.ns, name), &prometheus_v1.ServiceMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.ServiceMonitor), err
}

// List takes label and field selectors, and returns the list of ServiceMonitors that match those selectors.
func (c *FakeServiceMonitors) List(opts v1.ListOptions) (result *prometheus_v1.ServiceMonitorList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(servicemonitorsResource, servicemonitorsKind, c.ns, opts), &prometheus_v1.ServiceMonitorList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &prometheus_v1.ServiceMonitorList{}
	for _, item := range obj.(*prometheus_v1.ServiceMonitorList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested serviceMonitors.
func (c *FakeServiceMonitors) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(servicemonitorsResource, c.ns, opts))

}

// Create takes the representation of a serviceMonitor and creates it.  Returns the server's representation of the serviceMonitor, and an error, if there is any.
func (c *FakeServiceMonitors) Create(serviceMonitor *prometheus_v1.ServiceMonitor) (result *prometheus_v1.ServiceMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(servicemonitorsResource, c.ns, serviceMonitor), &prometheus_v1.ServiceMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.ServiceMonitor), err
}

// Update takes the representation of a serviceMonitor and updates it. Returns the server's representation of the serviceMonitor, and an error, if there is any.
func (c *FakeServiceMonitors) Update(serviceMonitor *prometheus_v1.ServiceMonitor) (result *prometheus_v1.ServiceMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(servicemonitorsResource, c.ns, serviceMonitor), &prometheus_v1.ServiceMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.ServiceMonitor), err
}

// Delete takes name of the serviceMonitor and deletes it. Returns an error if one occurs.
func (c *FakeServiceMonitors) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(servicemonitorsResource, c.ns, name), &prometheus_v1.ServiceMonitor{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeServiceMonitors) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(servicemonitorsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &prometheus_v1.ServiceMonitorList{})
	return err
}

// Patch applies the patch and returns the patched serviceMonitor.
func (c *FakeServiceMonitors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *prometheus_v1.ServiceMonitor, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(servicemonitorsResource, c.ns, name, data, subresources...), &prometheus_v1.ServiceMonitor{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.ServiceMonitor), err
}
