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

// EtcdBackupConfigsGetter has a method to return a EtcdBackupConfigInterface.
// A group's client should implement this interface.
type EtcdBackupConfigsGetter interface {
	EtcdBackupConfigs(namespace string) EtcdBackupConfigInterface
}

// EtcdBackupConfigInterface has methods to work with EtcdBackupConfig resources.
type EtcdBackupConfigInterface interface {
	Create(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.CreateOptions) (*v1.EtcdBackupConfig, error)
	Update(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.UpdateOptions) (*v1.EtcdBackupConfig, error)
	UpdateStatus(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.UpdateOptions) (*v1.EtcdBackupConfig, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.EtcdBackupConfig, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.EtcdBackupConfigList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.EtcdBackupConfig, err error)
	EtcdBackupConfigExpansion
}

// etcdBackupConfigs implements EtcdBackupConfigInterface
type etcdBackupConfigs struct {
	client rest.Interface
	ns     string
}

// newEtcdBackupConfigs returns a EtcdBackupConfigs
func newEtcdBackupConfigs(c *KubermaticV1Client, namespace string) *etcdBackupConfigs {
	return &etcdBackupConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the etcdBackupConfig, and returns the corresponding etcdBackupConfig object, and an error if there is any.
func (c *etcdBackupConfigs) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.EtcdBackupConfig, err error) {
	result = &v1.EtcdBackupConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of EtcdBackupConfigs that match those selectors.
func (c *etcdBackupConfigs) List(ctx context.Context, opts metav1.ListOptions) (result *v1.EtcdBackupConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.EtcdBackupConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested etcdBackupConfigs.
func (c *etcdBackupConfigs) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a etcdBackupConfig and creates it.  Returns the server's representation of the etcdBackupConfig, and an error, if there is any.
func (c *etcdBackupConfigs) Create(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.CreateOptions) (result *v1.EtcdBackupConfig, err error) {
	result = &v1.EtcdBackupConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(etcdBackupConfig).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a etcdBackupConfig and updates it. Returns the server's representation of the etcdBackupConfig, and an error, if there is any.
func (c *etcdBackupConfigs) Update(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.UpdateOptions) (result *v1.EtcdBackupConfig, err error) {
	result = &v1.EtcdBackupConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		Name(etcdBackupConfig.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(etcdBackupConfig).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *etcdBackupConfigs) UpdateStatus(ctx context.Context, etcdBackupConfig *v1.EtcdBackupConfig, opts metav1.UpdateOptions) (result *v1.EtcdBackupConfig, err error) {
	result = &v1.EtcdBackupConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		Name(etcdBackupConfig.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(etcdBackupConfig).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the etcdBackupConfig and deletes it. Returns an error if one occurs.
func (c *etcdBackupConfigs) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *etcdBackupConfigs) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched etcdBackupConfig.
func (c *etcdBackupConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.EtcdBackupConfig, err error) {
	result = &v1.EtcdBackupConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("etcdbackupconfigs").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
