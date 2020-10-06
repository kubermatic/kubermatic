// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	scheme "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ConstraintsGetter has a method to return a ConstraintInterface.
// A group's client should implement this interface.
type ConstraintsGetter interface {
	Constraints(namespace string) ConstraintInterface
}

// ConstraintInterface has methods to work with Constraint resources.
type ConstraintInterface interface {
	Create(ctx context.Context, constraint *v1.Constraint, opts metav1.CreateOptions) (*v1.Constraint, error)
	Update(ctx context.Context, constraint *v1.Constraint, opts metav1.UpdateOptions) (*v1.Constraint, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Constraint, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.ConstraintList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Constraint, err error)
	ConstraintExpansion
}

// constraints implements ConstraintInterface
type constraints struct {
	client rest.Interface
	ns     string
}

// newConstraints returns a Constraints
func newConstraints(c *KubermaticV1Client, namespace string) *constraints {
	return &constraints{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the constraint, and returns the corresponding constraint object, and an error if there is any.
func (c *constraints) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Constraint, err error) {
	result = &v1.Constraint{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("constraints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Constraints that match those selectors.
func (c *constraints) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ConstraintList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.ConstraintList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("constraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested constraints.
func (c *constraints) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("constraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a constraint and creates it.  Returns the server's representation of the constraint, and an error, if there is any.
func (c *constraints) Create(ctx context.Context, constraint *v1.Constraint, opts metav1.CreateOptions) (result *v1.Constraint, err error) {
	result = &v1.Constraint{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("constraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(constraint).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a constraint and updates it. Returns the server's representation of the constraint, and an error, if there is any.
func (c *constraints) Update(ctx context.Context, constraint *v1.Constraint, opts metav1.UpdateOptions) (result *v1.Constraint, err error) {
	result = &v1.Constraint{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("constraints").
		Name(constraint.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(constraint).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the constraint and deletes it. Returns an error if one occurs.
func (c *constraints) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("constraints").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *constraints) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("constraints").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched constraint.
func (c *constraints) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Constraint, err error) {
	result = &v1.Constraint{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("constraints").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
