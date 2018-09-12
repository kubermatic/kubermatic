package v1

import (
	scheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// UserProjectBindingsGetter has a method to return a UserProjectBindingInterface.
// A group's client should implement this interface.
type UserProjectBindingsGetter interface {
	UserProjectBindings() UserProjectBindingInterface
}

// UserProjectBindingInterface has methods to work with UserProjectBinding resources.
type UserProjectBindingInterface interface {
	Create(*v1.UserProjectBinding) (*v1.UserProjectBinding, error)
	Update(*v1.UserProjectBinding) (*v1.UserProjectBinding, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.UserProjectBinding, error)
	List(opts meta_v1.ListOptions) (*v1.UserProjectBindingList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.UserProjectBinding, err error)
	UserProjectBindingExpansion
}

// userProjectBindings implements UserProjectBindingInterface
type userProjectBindings struct {
	client rest.Interface
}

// newUserProjectBindings returns a UserProjectBindings
func newUserProjectBindings(c *KubermaticV1Client) *userProjectBindings {
	return &userProjectBindings{
		client: c.RESTClient(),
	}
}

// Get takes name of the userProjectBinding, and returns the corresponding userProjectBinding object, and an error if there is any.
func (c *userProjectBindings) Get(name string, options meta_v1.GetOptions) (result *v1.UserProjectBinding, err error) {
	result = &v1.UserProjectBinding{}
	err = c.client.Get().
		Resource("userprojectbindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of UserProjectBindings that match those selectors.
func (c *userProjectBindings) List(opts meta_v1.ListOptions) (result *v1.UserProjectBindingList, err error) {
	result = &v1.UserProjectBindingList{}
	err = c.client.Get().
		Resource("userprojectbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested userProjectBindings.
func (c *userProjectBindings) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("userprojectbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a userProjectBinding and creates it.  Returns the server's representation of the userProjectBinding, and an error, if there is any.
func (c *userProjectBindings) Create(userProjectBinding *v1.UserProjectBinding) (result *v1.UserProjectBinding, err error) {
	result = &v1.UserProjectBinding{}
	err = c.client.Post().
		Resource("userprojectbindings").
		Body(userProjectBinding).
		Do().
		Into(result)
	return
}

// Update takes the representation of a userProjectBinding and updates it. Returns the server's representation of the userProjectBinding, and an error, if there is any.
func (c *userProjectBindings) Update(userProjectBinding *v1.UserProjectBinding) (result *v1.UserProjectBinding, err error) {
	result = &v1.UserProjectBinding{}
	err = c.client.Put().
		Resource("userprojectbindings").
		Name(userProjectBinding.Name).
		Body(userProjectBinding).
		Do().
		Into(result)
	return
}

// Delete takes name of the userProjectBinding and deletes it. Returns an error if one occurs.
func (c *userProjectBindings) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("userprojectbindings").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *userProjectBindings) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("userprojectbindings").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched userProjectBinding.
func (c *userProjectBindings) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.UserProjectBinding, err error) {
	result = &v1.UserProjectBinding{}
	err = c.client.Patch(pt).
		Resource("userprojectbindings").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
