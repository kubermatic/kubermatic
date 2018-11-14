package v1alpha1

import (
	scheme "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// MachineClassesGetter has a method to return a MachineClassInterface.
// A group's client should implement this interface.
type MachineClassesGetter interface {
	MachineClasses(namespace string) MachineClassInterface
}

// MachineClassInterface has methods to work with MachineClass resources.
type MachineClassInterface interface {
	Create(*v1alpha1.MachineClass) (*v1alpha1.MachineClass, error)
	Update(*v1alpha1.MachineClass) (*v1alpha1.MachineClass, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MachineClass, error)
	List(opts v1.ListOptions) (*v1alpha1.MachineClassList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MachineClass, err error)
	MachineClassExpansion
}

// machineClasses implements MachineClassInterface
type machineClasses struct {
	client rest.Interface
	ns     string
}

// newMachineClasses returns a MachineClasses
func newMachineClasses(c *ClusterV1alpha1Client, namespace string) *machineClasses {
	return &machineClasses{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the machineClass, and returns the corresponding machineClass object, and an error if there is any.
func (c *machineClasses) Get(name string, options v1.GetOptions) (result *v1alpha1.MachineClass, err error) {
	result = &v1alpha1.MachineClass{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machineclasses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MachineClasses that match those selectors.
func (c *machineClasses) List(opts v1.ListOptions) (result *v1alpha1.MachineClassList, err error) {
	result = &v1alpha1.MachineClassList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("machineclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested machineClasses.
func (c *machineClasses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("machineclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a machineClass and creates it.  Returns the server's representation of the machineClass, and an error, if there is any.
func (c *machineClasses) Create(machineClass *v1alpha1.MachineClass) (result *v1alpha1.MachineClass, err error) {
	result = &v1alpha1.MachineClass{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("machineclasses").
		Body(machineClass).
		Do().
		Into(result)
	return
}

// Update takes the representation of a machineClass and updates it. Returns the server's representation of the machineClass, and an error, if there is any.
func (c *machineClasses) Update(machineClass *v1alpha1.MachineClass) (result *v1alpha1.MachineClass, err error) {
	result = &v1alpha1.MachineClass{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("machineclasses").
		Name(machineClass.Name).
		Body(machineClass).
		Do().
		Into(result)
	return
}

// Delete takes name of the machineClass and deletes it. Returns an error if one occurs.
func (c *machineClasses) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machineclasses").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *machineClasses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("machineclasses").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched machineClass.
func (c *machineClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MachineClass, err error) {
	result = &v1alpha1.MachineClass{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("machineclasses").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
