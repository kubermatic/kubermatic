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

// FakePrometheuses implements PrometheusInterface
type FakePrometheuses struct {
	Fake *FakeMonitoringV1
	ns   string
}

var prometheusesResource = schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheuses"}

var prometheusesKind = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "Prometheus"}

// Get takes name of the prometheus, and returns the corresponding prometheus object, and an error if there is any.
func (c *FakePrometheuses) Get(name string, options v1.GetOptions) (result *prometheus_v1.Prometheus, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(prometheusesResource, c.ns, name), &prometheus_v1.Prometheus{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.Prometheus), err
}

// List takes label and field selectors, and returns the list of Prometheuses that match those selectors.
func (c *FakePrometheuses) List(opts v1.ListOptions) (result *prometheus_v1.PrometheusList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(prometheusesResource, prometheusesKind, c.ns, opts), &prometheus_v1.PrometheusList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &prometheus_v1.PrometheusList{}
	for _, item := range obj.(*prometheus_v1.PrometheusList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested prometheuses.
func (c *FakePrometheuses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(prometheusesResource, c.ns, opts))

}

// Create takes the representation of a prometheus and creates it.  Returns the server's representation of the prometheus, and an error, if there is any.
func (c *FakePrometheuses) Create(prometheus *prometheus_v1.Prometheus) (result *prometheus_v1.Prometheus, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(prometheusesResource, c.ns, prometheus), &prometheus_v1.Prometheus{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.Prometheus), err
}

// Update takes the representation of a prometheus and updates it. Returns the server's representation of the prometheus, and an error, if there is any.
func (c *FakePrometheuses) Update(prometheus *prometheus_v1.Prometheus) (result *prometheus_v1.Prometheus, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(prometheusesResource, c.ns, prometheus), &prometheus_v1.Prometheus{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.Prometheus), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePrometheuses) UpdateStatus(prometheus *prometheus_v1.Prometheus) (*prometheus_v1.Prometheus, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(prometheusesResource, "status", c.ns, prometheus), &prometheus_v1.Prometheus{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.Prometheus), err
}

// Delete takes name of the prometheus and deletes it. Returns an error if one occurs.
func (c *FakePrometheuses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(prometheusesResource, c.ns, name), &prometheus_v1.Prometheus{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePrometheuses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(prometheusesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &prometheus_v1.PrometheusList{})
	return err
}

// Patch applies the patch and returns the patched prometheus.
func (c *FakePrometheuses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *prometheus_v1.Prometheus, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(prometheusesResource, c.ns, name, data, subresources...), &prometheus_v1.Prometheus{})

	if obj == nil {
		return nil, err
	}
	return obj.(*prometheus_v1.Prometheus), err
}
