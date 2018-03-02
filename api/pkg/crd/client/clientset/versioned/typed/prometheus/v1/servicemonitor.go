package v1

import (
	scheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ServiceMonitorsGetter has a method to return a ServiceMonitorInterface.
// A group's client should implement this interface.
type ServiceMonitorsGetter interface {
	ServiceMonitors(namespace string) ServiceMonitorInterface
}

// ServiceMonitorInterface has methods to work with ServiceMonitor resources.
type ServiceMonitorInterface interface {
	Create(*v1.ServiceMonitor) (*v1.ServiceMonitor, error)
	Update(*v1.ServiceMonitor) (*v1.ServiceMonitor, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.ServiceMonitor, error)
	List(opts meta_v1.ListOptions) (*v1.ServiceMonitorList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ServiceMonitor, err error)
	ServiceMonitorExpansion
}

// serviceMonitors implements ServiceMonitorInterface
type serviceMonitors struct {
	client rest.Interface
	ns     string
}

// newServiceMonitors returns a ServiceMonitors
func newServiceMonitors(c *MonitoringV1Client, namespace string) *serviceMonitors {
	return &serviceMonitors{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the serviceMonitor, and returns the corresponding serviceMonitor object, and an error if there is any.
func (c *serviceMonitors) Get(name string, options meta_v1.GetOptions) (result *v1.ServiceMonitor, err error) {
	result = &v1.ServiceMonitor{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("servicemonitors").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ServiceMonitors that match those selectors.
func (c *serviceMonitors) List(opts meta_v1.ListOptions) (result *v1.ServiceMonitorList, err error) {
	result = &v1.ServiceMonitorList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("servicemonitors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested serviceMonitors.
func (c *serviceMonitors) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("servicemonitors").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a serviceMonitor and creates it.  Returns the server's representation of the serviceMonitor, and an error, if there is any.
func (c *serviceMonitors) Create(serviceMonitor *v1.ServiceMonitor) (result *v1.ServiceMonitor, err error) {
	result = &v1.ServiceMonitor{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("servicemonitors").
		Body(serviceMonitor).
		Do().
		Into(result)
	return
}

// Update takes the representation of a serviceMonitor and updates it. Returns the server's representation of the serviceMonitor, and an error, if there is any.
func (c *serviceMonitors) Update(serviceMonitor *v1.ServiceMonitor) (result *v1.ServiceMonitor, err error) {
	result = &v1.ServiceMonitor{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("servicemonitors").
		Name(serviceMonitor.Name).
		Body(serviceMonitor).
		Do().
		Into(result)
	return
}

// Delete takes name of the serviceMonitor and deletes it. Returns an error if one occurs.
func (c *serviceMonitors) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("servicemonitors").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *serviceMonitors) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("servicemonitors").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched serviceMonitor.
func (c *serviceMonitors) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.ServiceMonitor, err error) {
	result = &v1.ServiceMonitor{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("servicemonitors").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
