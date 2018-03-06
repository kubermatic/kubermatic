package v1

import (
	scheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// UserSSHKeiesGetter has a method to return a UserSSHKeyInterface.
// A group's client should implement this interface.
type UserSSHKeiesGetter interface {
	UserSSHKeies() UserSSHKeyInterface
}

// UserSSHKeyInterface has methods to work with UserSSHKey resources.
type UserSSHKeyInterface interface {
	Create(*v1.UserSSHKey) (*v1.UserSSHKey, error)
	Update(*v1.UserSSHKey) (*v1.UserSSHKey, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.UserSSHKey, error)
	List(opts meta_v1.ListOptions) (*v1.UserSSHKeyList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.UserSSHKey, err error)
	UserSSHKeyExpansion
}

// userSSHKeies implements UserSSHKeyInterface
type userSSHKeies struct {
	client rest.Interface
}

// newUserSSHKeies returns a UserSSHKeies
func newUserSSHKeies(c *KubermaticV1Client) *userSSHKeies {
	return &userSSHKeies{
		client: c.RESTClient(),
	}
}

// Get takes name of the userSSHKey, and returns the corresponding userSSHKey object, and an error if there is any.
func (c *userSSHKeies) Get(name string, options meta_v1.GetOptions) (result *v1.UserSSHKey, err error) {
	result = &v1.UserSSHKey{}
	err = c.client.Get().
		Resource("usersshkeies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of UserSSHKeies that match those selectors.
func (c *userSSHKeies) List(opts meta_v1.ListOptions) (result *v1.UserSSHKeyList, err error) {
	result = &v1.UserSSHKeyList{}
	err = c.client.Get().
		Resource("usersshkeies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested userSSHKeies.
func (c *userSSHKeies) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("usersshkeies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a userSSHKey and creates it.  Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *userSSHKeies) Create(userSSHKey *v1.UserSSHKey) (result *v1.UserSSHKey, err error) {
	result = &v1.UserSSHKey{}
	err = c.client.Post().
		Resource("usersshkeies").
		Body(userSSHKey).
		Do().
		Into(result)
	return
}

// Update takes the representation of a userSSHKey and updates it. Returns the server's representation of the userSSHKey, and an error, if there is any.
func (c *userSSHKeies) Update(userSSHKey *v1.UserSSHKey) (result *v1.UserSSHKey, err error) {
	result = &v1.UserSSHKey{}
	err = c.client.Put().
		Resource("usersshkeies").
		Name(userSSHKey.Name).
		Body(userSSHKey).
		Do().
		Into(result)
	return
}

// Delete takes name of the userSSHKey and deletes it. Returns an error if one occurs.
func (c *userSSHKeies) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("usersshkeies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *userSSHKeies) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("usersshkeies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched userSSHKey.
func (c *userSSHKeies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.UserSSHKey, err error) {
	result = &v1.UserSSHKey{}
	err = c.client.Patch(pt).
		Resource("usersshkeies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
