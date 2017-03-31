package extensions

import (
	"fmt"
	"regexp"
	"strings"

	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/selection"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
)

var replace = regexp.MustCompile(`[^a-z0-9]*`)

// NormalizeUser is the base64 k8s compatible representation for a user
func NormalizeUser(s string) string {
	s = strings.ToLower(s)
	return replace.ReplaceAllString(s, "")
}

// ConstructSerialKeyName generates a name for a serial key which is accepted by k8s metadata.Name
func ConstructSerialKeyName(username, fingerprint string) string {
	return fmt.Sprintf("%s-%s", NormalizeUser(username), strings.NewReplacer(":", "").Replace(fingerprint))
}

// WrapClientsetWithExtensions returns a clientset to work with extensions
func WrapClientsetWithExtensions(config *rest.Config) (Clientset, error) {
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
	SSHKeyTPR(user string) SSHKeyTPRInterface
	Nodes(ns string) NodesInterface
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

// SSHKeyTPR returns an interface to interact with UserSSHKey
func (w *WrappedClientset) SSHKeyTPR(user string) SSHKeyTPRInterface {
	return &SSHKeyTPRClient{
		client: w.Client,
		user:   user,
	}
}

// Nodes returns an interface to interact with Nodes
func (w *WrappedClientset) Nodes(ns string) NodesInterface {
	return &NodesClient{
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

// SSHKeyTPRInterface is the interface for an SSHTPR client
type SSHKeyTPRInterface interface {
	Create(*UserSSHKey) (*UserSSHKey, error)
	List() (UserSSHKeyList, error)
	Delete(fingerprint string, options *v1.DeleteOptions) error
}

// SSHKeyTPRClient is an implementation of SSHKeyTPRInterface to work with stored SSH keys
type SSHKeyTPRClient struct {
	client rest.Interface
	user   string
}

func (s *SSHKeyTPRClient) injectUserLabel(sk *UserSSHKey) {
	lbs := sk.Metadata.Labels
	if lbs == nil {
		lbs = map[string]string{}
	}
	lbs["user"] = NormalizeUser(s.user)
	sk.Metadata.SetLabels(lbs)
}

// Create saves an SSHKey into an tpr
func (s *SSHKeyTPRClient) Create(sk *UserSSHKey) (*UserSSHKey, error) {
	var result UserSSHKey
	s.injectUserLabel(sk)
	err := s.client.Post().
		Namespace(SSHKeyTPRNamespace).
		Resource(SSHKeyTPRName).
		Body(sk).
		Do().
		Into(&result)
	return &result, err
}

// List returns all SSHKey's for a given User
func (s *SSHKeyTPRClient) List() (UserSSHKeyList, error) {
	opts := v1.ListOptions{}
	label, err := labels.NewRequirement("user", selection.Equals, []string{NormalizeUser(s.user)})
	if err != nil {
		return UserSSHKeyList{}, err
	}
	var result UserSSHKeyList
	err = s.client.Get().
		Namespace(SSHKeyTPRNamespace).
		Resource(SSHKeyTPRName).
		VersionedParams(&opts, kapi.ParameterCodec).
		LabelsSelectorParam(labels.NewSelector().Add(*label)).
		Do().
		Into(&result)

	return result, err

}

// Delete takes the fingerprint of the ssh key and deletes it. Returns an error if one occurs.
func (s *SSHKeyTPRClient) Delete(fingerprint string, options *v1.DeleteOptions) error {
	return s.client.Delete().
		Namespace(SSHKeyTPRNamespace).
		Resource(SSHKeyTPRName).
		Name(ConstructSerialKeyName(s.user, fingerprint)).
		// TODO: workaround, remove this when delete options are allowed
		Body(options).
		Do().
		Error()
}

// NodesInterface is an interface to interact with ClNode TPRs
type NodesInterface interface {
	Create(*ClNode) (*ClNode, error)
	Get(name string) (*ClNode, error)
	List(v1.ListOptions) (*ClNodeList, error)
	Watch(v1.ListOptions) (watch.Interface, error)
	Update(*ClNode) (*ClNode, error)
	Delete(string, *v1.DeleteOptions) error
}

// NodesClient is an implementation of NodesInterface to work with Nodes
type NodesClient struct {
	client rest.Interface
	ns     string
}

// Create makes a new node in the node TPR or returns an existing one with an error.
func (c *NodesClient) Create(node *ClNode) (*ClNode, error) {
	result := &ClNode{}
	err := c.client.Post().
		Namespace(c.ns).
		Resource(NodeTPRName).
		Body(node).
		Do().
		Into(result)
	return result, err
}

// List takes list options and returns a list of nodes.
func (c *NodesClient) List(opts v1.ListOptions) (*ClNodeList, error) {
	result := &ClNodeList{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource(NodeTPRName).
		VersionedParams(&opts, kapi.ParameterCodec).
		Do().
		Into(result)
	return result, err
}

// Watch returns a watch.Interface that watches the requested node
func (c *NodesClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(c.ns).
		Prefix("watch").
		Resource(NodeTPRName).
		VersionedParams(&opts, kapi.ParameterCodec).
		Watch()
}

// Update ..... updates a given node dahhh
func (c *NodesClient) Update(node *ClNode) (*ClNode, error) {
	result := &ClNode{}
	err := c.client.Put().
		Namespace(c.ns).
		Resource(NodeTPRName).
		Name(node.Metadata.Name).
		Body(node).
		Do().
		Into(result)
	return result, err
}

// Delete takes the name of a node and removes it from the TPR
func (c *NodesClient) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(NodeTPRName).
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get takes the name of a node and fetches it from the TPR.
func (c *NodesClient) Get(name string) (*ClNode, error) {
	result := &ClNode{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource(NodeTPRName).
		Name(name).
		Do().
		Into(result)
	return result, err
}
