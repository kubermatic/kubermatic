package extensions

import (
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
)

// WrapClientsetWithExtensions returns a clientset to work with extensions
func WrapClientsetWithExtensions(config *rest.Config) (*WrappedClientset, error) {
	restConfig := &rest.Config{}
	*restConfig = *config
	c, err := extensionClient(restConfig)
	if err != nil {
		return nil, err
	}
	return &WrappedClientset{
		Client: c,
	}, nil
}

func extensionClient(config *rest.Config) (*rest.RESTClient, error) {
	config.APIPath = "/apis"
	config.ContentConfig = rest.ContentConfig{
		GroupVersion: &schema.GroupVersion{
			Group:   GroupName,
			Version: Version,
		},
		NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: kapi.Codecs},
		ContentType:          runtime.ContentTypeJSON,
	}

	return rest.RESTClientFor(config)
}

// Clientset is an interface to work with extensions
type Clientset interface {
	ClusterAddons(ns string) ClusterAddonsInterface
}

// WrappedClientset is an implementation of the ExtensionsClientset interface to work with extensions
type WrappedClientset struct {
	Client *rest.RESTClient
}

// ClusterAddons returns an interface to interact with ClusterAddons
func (w *WrappedClientset) ClusterAddons(ns string) ClusterAddonsInterface {
	return &ClusterAddonsClient{
		client: w.Client,
		ns:     ns,
	}
}

// ClusterAddonsInterface is an interface to interact with ClusterAddons
type ClusterAddonsInterface interface {
	Create(*ClusterAddon) (*ClusterAddon, error)
	Get(name string) (*ClusterAddon, error)
	List(v1.ListOptions) (*ClusterAddonList, error)
	Watch(v1.ListOptions) (watch.Interface, error)
	Update(*ClusterAddon) (*ClusterAddon, error)
	Delete(string, *v1.DeleteOptions) error
}

// ClusterAddonsClient is an implementation of ClusterAddonsInterface to work with ClusterAddons
type ClusterAddonsClient struct {
	client rest.Interface
	ns     string
}

// Create takes the representation of a cluster addon and creates it.  Returns the server's representation of the cluster addon, and an error, if there is any.
func (c *ClusterAddonsClient) Create(addon *ClusterAddon) (result *ClusterAddon, err error) {
	result = &ClusterAddon{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("clusteraddons").
		Body(addon).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterAddons that match those selectors.
func (c *ClusterAddonsClient) List(opts v1.ListOptions) (result *ClusterAddonList, err error) {
	result = &ClusterAddonList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusteraddons").
		VersionedParams(&opts, kapi.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cluster addons.
func (c *ClusterAddonsClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(c.ns).
		Prefix("watch").
		Resource("clusteraddons").
		VersionedParams(&opts, kapi.ParameterCodec).
		Watch()
}

// Update takes the representation of a cluster addon and updates it. Returns the server's representation of the cluster addon, and an error, if there is any.
func (c *ClusterAddonsClient) Update(addon *ClusterAddon) (result *ClusterAddon, err error) {
	result = &ClusterAddon{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("clusteraddons").
		Name(addon.Metadata.Name).
		Body(addon).
		Do().
		Into(result)
	return
}

// Delete takes name of the cluster addon and deletes it. Returns an error if one occurs.
func (c *ClusterAddonsClient) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("clusteraddons").
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get takes name of the cluster addon, and returns the corresponding cluster addon object, and an error if there is any.
func (c *ClusterAddonsClient) Get(name string) (result *ClusterAddon, err error) {
	result = &ClusterAddon{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusteraddons").
		Name(name).
		Do().
		Into(result)
	return
}

type SSHKeyTPRInterface interface {
	Create(*SSHKey) (*SSHKey, error)
	List() ([]*SSHKey, error)
	Delete(fingerprint string, options *v1.DeleteOptions) error
}
