/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	scheme "github.com/kube-node/nodeset/pkg/client/clientset/versioned/scheme"
	v1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NodeClassesGetter has a method to return a NodeClassInterface.
// A group's client should implement this interface.
type NodeClassesGetter interface {
	NodeClasses() NodeClassInterface
}

// NodeClassInterface has methods to work with NodeClass resources.
type NodeClassInterface interface {
	Create(*v1alpha1.NodeClass) (*v1alpha1.NodeClass, error)
	Update(*v1alpha1.NodeClass) (*v1alpha1.NodeClass, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.NodeClass, error)
	List(opts v1.ListOptions) (*v1alpha1.NodeClassList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeClass, err error)
	NodeClassExpansion
}

// nodeClasses implements NodeClassInterface
type nodeClasses struct {
	client rest.Interface
}

// newNodeClasses returns a NodeClasses
func newNodeClasses(c *NodesetV1alpha1Client) *nodeClasses {
	return &nodeClasses{
		client: c.RESTClient(),
	}
}

// Get takes name of the nodeClass, and returns the corresponding nodeClass object, and an error if there is any.
func (c *nodeClasses) Get(name string, options v1.GetOptions) (result *v1alpha1.NodeClass, err error) {
	result = &v1alpha1.NodeClass{}
	err = c.client.Get().
		Resource("nodeclasses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NodeClasses that match those selectors.
func (c *nodeClasses) List(opts v1.ListOptions) (result *v1alpha1.NodeClassList, err error) {
	result = &v1alpha1.NodeClassList{}
	err = c.client.Get().
		Resource("nodeclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nodeClasses.
func (c *nodeClasses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("nodeclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a nodeClass and creates it.  Returns the server's representation of the nodeClass, and an error, if there is any.
func (c *nodeClasses) Create(nodeClass *v1alpha1.NodeClass) (result *v1alpha1.NodeClass, err error) {
	result = &v1alpha1.NodeClass{}
	err = c.client.Post().
		Resource("nodeclasses").
		Body(nodeClass).
		Do().
		Into(result)
	return
}

// Update takes the representation of a nodeClass and updates it. Returns the server's representation of the nodeClass, and an error, if there is any.
func (c *nodeClasses) Update(nodeClass *v1alpha1.NodeClass) (result *v1alpha1.NodeClass, err error) {
	result = &v1alpha1.NodeClass{}
	err = c.client.Put().
		Resource("nodeclasses").
		Name(nodeClass.Name).
		Body(nodeClass).
		Do().
		Into(result)
	return
}

// Delete takes name of the nodeClass and deletes it. Returns an error if one occurs.
func (c *nodeClasses) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("nodeclasses").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nodeClasses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("nodeclasses").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched nodeClass.
func (c *nodeClasses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.NodeClass, err error) {
	result = &v1alpha1.NodeClass{}
	err = c.client.Patch(pt).
		Resource("nodeclasses").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
