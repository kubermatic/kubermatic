// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	scheme "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterTemplatesGetter has a method to return a ClusterTemplateInterface.
// A group's client should implement this interface.
type ClusterTemplatesGetter interface {
	ClusterTemplates() ClusterTemplateInterface
}

// ClusterTemplateInterface has methods to work with ClusterTemplate resources.
type ClusterTemplateInterface interface {
	Create(ctx context.Context, clusterTemplate *v1.ClusterTemplate, opts metav1.CreateOptions) (*v1.ClusterTemplate, error)
	Update(ctx context.Context, clusterTemplate *v1.ClusterTemplate, opts metav1.UpdateOptions) (*v1.ClusterTemplate, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.ClusterTemplate, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ClusterTemplateList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterTemplate, err error)
	ClusterTemplateExpansion
}

// clusterTemplates implements ClusterTemplateInterface
type clusterTemplates struct {
	client rest.Interface
}

// newClusterTemplates returns a ClusterTemplates
func newClusterTemplates(c *KubermaticV1Client) *clusterTemplates {
	return &clusterTemplates{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterTemplate, and returns the corresponding clusterTemplate object, and an error if there is any.
func (c *clusterTemplates) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ClusterTemplate, err error) {
	result = &v1.ClusterTemplate{}
	err = c.client.Get().
		Resource("clustertemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterTemplates that match those selectors.
func (c *clusterTemplates) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ClusterTemplateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ClusterTemplateList{}
	err = c.client.Get().
		Resource("clustertemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterTemplates.
func (c *clusterTemplates) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clustertemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterTemplate and creates it.  Returns the server's representation of the clusterTemplate, and an error, if there is any.
func (c *clusterTemplates) Create(ctx context.Context, clusterTemplate *v1.ClusterTemplate, opts metav1.CreateOptions) (result *v1.ClusterTemplate, err error) {
	result = &v1.ClusterTemplate{}
	err = c.client.Post().
		Resource("clustertemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterTemplate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterTemplate and updates it. Returns the server's representation of the clusterTemplate, and an error, if there is any.
func (c *clusterTemplates) Update(ctx context.Context, clusterTemplate *v1.ClusterTemplate, opts metav1.UpdateOptions) (result *v1.ClusterTemplate, err error) {
	result = &v1.ClusterTemplate{}
	err = c.client.Put().
		Resource("clustertemplates").
		Name(clusterTemplate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterTemplate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterTemplate and deletes it. Returns an error if one occurs.
func (c *clusterTemplates) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clustertemplates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterTemplates) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clustertemplates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterTemplate.
func (c *clusterTemplates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ClusterTemplate, err error) {
	result = &v1.ClusterTemplate{}
	err = c.client.Patch(pt).
		Resource("clustertemplates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
