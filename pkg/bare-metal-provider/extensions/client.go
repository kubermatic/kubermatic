package extensions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	kapi "k8s.io/client-go/pkg/api"
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
	Nodes(ns string) NodesInterface
	Clusters(ns string) ClusterInterface
}

// WrappedClientset is an implementation of the ExtensionsClientset interface to work with extensions
type WrappedClientset struct {
	Client *rest.RESTClient
}

// Nodes returns an interface to interact with nodes
func (w *WrappedClientset) Nodes(ns string) NodesInterface {
	return &NodesClient{
		client: w.Client,
		ns:     ns,
	}
}

// Clusters returns an interface to interact with clusters
func (w *WrappedClientset) Clusters(ns string) ClusterInterface {
	return &ClusterClient{
		client: w.Client,
		ns:     ns,
	}
}

// ClusterInterface is an interface to interact with clusters
type ClusterInterface interface {
	Create(*Cluster) (*Cluster, error)
	Get(name string) (*Cluster, error)
	List(metav1.ListOptions) (*ClusterList, error)
	Watch(metav1.ListOptions) (watch.Interface, error)
	Update(*Cluster) (*Cluster, error)
	Delete(string, *metav1.DeleteOptions) error
}

// NodesInterface is an interface to interact with Nodes
type NodesInterface interface {
	Create(*Node) (*Node, error)
	Get(name string) (*Node, error)
	List(metav1.ListOptions) (*NodeList, error)
	Watch(options metav1.ListOptions) (watch.Interface, error)
	Update(*Node) (*Node, error)
	Delete(string, *metav1.DeleteOptions) error
}

// NodesClient is an implementation of NodesInterface to work with Nodes
type NodesClient struct {
	client rest.Interface
	ns     string
}

// ClusterClient is an implementation of ClusterInterface to work with clusters
type ClusterClient struct {
	client rest.Interface
	ns     string
}

// Create takes the representation of a node and creates it.  Returns the server's representation of the node, and an error, if there is any.
func (c *NodesClient) Create(node *Node) (result *Node, err error) {
	result = &Node{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource(NodeResourceName).
		Body(node).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Nodes that match those selectors.
func (c *NodesClient) List(opts metav1.ListOptions) (result *NodeList, err error) {
	result = &NodeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(NodeResourceName).
		VersionedParams(&opts, metav1.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nodes.
func (c *NodesClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(c.ns).
		Prefix("watch").
		Resource(NodeResourceName).
		VersionedParams(&opts, metav1.ParameterCodec).
		Watch()
}

// Update takes the representation of a node and updates it. Returns the server's representation of the node, and an error, if there is any.
func (c *NodesClient) Update(node *Node) (result *Node, err error) {
	result = &Node{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource(NodeResourceName).
		Name(node.Metadata.Name).
		Body(node).
		Do().
		Into(result)
	return
}

// Delete takes name(node-id) of the node and deletes it. Returns an error if one occurs.
func (c *NodesClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(NodeResourceName).
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get takes name(node-id) of the node, and returns the corresponding node object, and an error if there is any.
func (c *NodesClient) Get(name string) (result *Node, err error) {
	result = &Node{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(NodeResourceName).
		Name(name).
		Do().
		Into(result)
	return
}

// Create takes the representation of a cluster and creates it.  Returns the server's representation of the cluster, and an error, if there is any.
func (c *ClusterClient) Create(cluster *Cluster) (result *Cluster, err error) {
	result = &Cluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource(ClusterResourceName).
		Body(cluster).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of clusters that match those selectors.
func (c *ClusterClient) List(opts metav1.ListOptions) (result *ClusterList, err error) {
	result = &ClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(ClusterResourceName).
		VersionedParams(&opts, metav1.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusters.
func (c *ClusterClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(c.ns).
		Prefix("watch").
		Resource(ClusterResourceName).
		VersionedParams(&opts, metav1.ParameterCodec).
		Watch()
}

// Update takes the representation of a cluster and updates it. Returns the server's representation of the cluster, and an error, if there is any.
func (c *ClusterClient) Update(cluster *Cluster) (result *Cluster, err error) {
	result = &Cluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource(ClusterResourceName).
		Name(cluster.Metadata.Name).
		Body(cluster).
		Do().
		Into(result)
	return
}

// Delete takes name(cluster-id) of the cluster and deletes it. Returns an error if one occurs.
func (c *ClusterClient) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(ClusterResourceName).
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get takes name of the cluster, and returns the corresponding cluster object, and an error if there is any.
func (c *ClusterClient) Get(name string) (result *Cluster, err error) {
	result = &Cluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource(ClusterResourceName).
		Name(name).
		Do().
		Into(result)
	return
}
