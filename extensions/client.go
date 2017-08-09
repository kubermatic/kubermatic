package extensions

import (
	"fmt"
	"strings"

	"crypto/sha1"

	"encoding/base64"

	"github.com/kubermatic/api/uuid"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/selection"
	watch "k8s.io/apimachinery/pkg/watch"
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
)

// ConstructNewSerialKeyName generates a name for a serial key which is accepted by k8s metadata.Name
// Fingerprint is without colons
func ConstructNewSerialKeyName(fingerprint string) string {
	return fmt.Sprintf("key-%s-%s", fingerprint, uuid.ShortUID(4))
}

// NormalizeFingerprint returns a normalized fingerprint
func NormalizeFingerprint(f string) string {
	return strings.NewReplacer(":", "").Replace(f)
}

// NormalizeUser base64 encodes a user to store him in labels
// This is a one way function
func NormalizeUser(name string) string {
	// This part has to stay for backwards capability.
	// It we need this for old clusters which use an auth provider with useres, which will encode
	// in less then 63 chars.
	b := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(name))
	if len(b) <= 63 {
		return b
	}

	// This is the new way.
	// We can use a weak hash because we trust the authority, which generates the name.
	// This will always yield a string which makes the user identifiable and is less than 63 chars
	// due to the usage of a hash function.
	// Potentially we could have collisions, but this is not avoidable, due to the fact that the
	// set of our domain is smaller than our codomain.
	// It's trivial to see that we can't reverse this due to the fact that our function is not injective. q.e.d
	sh := sha1.New()
	fmt.Fprint(sh, name)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
}

// GenerateNormalizedFigerprint a normalized fingerprint from a public key
func GenerateNormalizedFigerprint(pub string) (string, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pub))
	if err != nil {
		return "", err
	}
	return ssh.FingerprintLegacyMD5(pubKey), nil
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

	v1.AddToGroupVersion(kapi.Scheme, SchemeGroupVersion)
	if err := SchemeBuilder.AddToScheme(kapi.Scheme); err != nil {
		return nil, err
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
	Client rest.Interface
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
	Update(key *UserSSHKey) (result *UserSSHKey, err error)
}

// SSHKeyTPRClient is an implementation of SSHKeyTPRInterface to work with stored SSH keys
type SSHKeyTPRClient struct {
	client rest.Interface
	user   string
}

// Create saves an SSHKey into an tpr
func (s *SSHKeyTPRClient) Create(sk *UserSSHKey) (*UserSSHKey, error) {
	sk.addLabel("user", NormalizeUser(s.user))

	var result UserSSHKey
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

// Update takes the representation of a ssh key and updates it. Returns the server's representation of the ssh key, and an error, if there is any.
func (s *SSHKeyTPRClient) Update(key *UserSSHKey) (result *UserSSHKey, err error) {
	result = &UserSSHKey{}
	err = s.client.Put().
		Namespace(SSHKeyTPRNamespace).
		Resource(SSHKeyTPRName).
		Name(key.Metadata.Name).
		Body(key).
		Do().
		Into(result)
	return
}

// Delete takes the fingerprint of the ssh key and deletes it. Returns an error if one occurs.
func (s *SSHKeyTPRClient) Delete(resourceName string, options *v1.DeleteOptions) error {
	return s.client.Delete().
		Namespace(SSHKeyTPRNamespace).
		Resource(SSHKeyTPRName).
		Name(resourceName).
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
