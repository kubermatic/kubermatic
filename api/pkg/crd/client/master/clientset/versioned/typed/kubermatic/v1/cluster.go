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

package v1

import (
	scheme "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/scheme"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClustersGetter has a method to return a ClusterInterface.
// A group's client should implement this interface.
type ClustersGetter interface {
	Clusters() ClusterInterface
}

// ClusterInterface has methods to work with Cluster resources.
type ClusterInterface interface {
	Create(*v1.Cluster) (*v1.Cluster, error)
	Update(*v1.Cluster) (*v1.Cluster, error)
	UpdateStatus(*v1.Cluster) (*v1.Cluster, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.Cluster, error)
	List(opts meta_v1.ListOptions) (*v1.ClusterList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Cluster, err error)
	ClusterExpansion
}

// clusters implements ClusterInterface
type clusters struct {
	client rest.Interface
}

// newClusters returns a Clusters
func newClusters(c *KubermaticV1Client) *clusters {
	return &clusters{
		client: c.RESTClient(),
	}
}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *clusters) Get(name string, options meta_v1.GetOptions) (result *v1.Cluster, err error) {
	result = &v1.Cluster{}
	err = c.client.Get().
		Resource("clusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Clusters that match those selectors.
func (c *clusters) List(opts meta_v1.ListOptions) (result *v1.ClusterList, err error) {
	result = &v1.ClusterList{}
	err = c.client.Get().
		Resource("clusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *clusters) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("clusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *clusters) Create(cluster *v1.Cluster) (result *v1.Cluster, err error) {
	result = &v1.Cluster{}
	err = c.client.Post().
		Resource("clusters").
		Body(cluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *clusters) Update(cluster *v1.Cluster) (result *v1.Cluster, err error) {
	result = &v1.Cluster{}
	err = c.client.Put().
		Resource("clusters").
		Name(cluster.Name).
		Body(cluster).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *clusters) UpdateStatus(cluster *v1.Cluster) (result *v1.Cluster, err error) {
	result = &v1.Cluster{}
	err = c.client.Put().
		Resource("clusters").
		Name(cluster.Name).
		SubResource("status").
		Body(cluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the cluster and deletes it. Returns an error if one occurs.
func (c *clusters) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusters) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Resource("clusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched cluster.
func (c *clusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Cluster, err error) {
	result = &v1.Cluster{}
	err = c.client.Patch(pt).
		Resource("clusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
