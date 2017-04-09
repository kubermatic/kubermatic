package etcd

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/kubermatic/api/uuid"
	"golang.org/x/crypto/ssh"
	kapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/watch"
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
func NormalizeUser(name string) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(name))
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
	c, err := etcdClusterClient(restConfig)
	if err != nil {
		return nil, err
	}
	return &WrappedClientset{
		Client: c,
	}, nil
}

func etcdClusterClient(config *rest.Config) (*rest.RESTClient, error) {
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
	Cluster(ns string) ClusterInterface
}

// WrappedClientset is an implementation of the ExtensionsClientset interface to work with extensions
type WrappedClientset struct {
	Client *rest.RESTClient
}

// Cluster returns an interface to interact with Cluster
func (w *WrappedClientset) Cluster(ns string) ClusterInterface {
	return &ClusterClient{
		client: w.Client,
		ns:     ns,
	}
}

// ClusterInterface is an interface to interact with Cluster Operator TPRs
type ClusterInterface interface {
	Create(*Cluster) (*Cluster, error)
	Get(name string) (*Cluster, error)
	List(v1.ListOptions) (*ClusterList, error)
	Watch(v1.ListOptions) (watch.Interface, error)
	Update(*Cluster) (*Cluster, error)
	Delete(string, *v1.DeleteOptions) error
}

// ClusterClient is an implementation of EtcdOperatorInterface to work with etcd-operator
type ClusterClient struct {
	client rest.Interface
	ns     string
}

// Create makes a new etcd-cluster in the or returns an existing one with an error.
func (c *ClusterClient) Create(etcd *Cluster) (*Cluster, error) {
	result := &Cluster{}
	err := c.client.Post().
		Namespace(c.ns).
		Resource(TPRKind).
		Body(etcd).
		Do().
		Into(result)
	return result, err
}

// List takes list options and returns a list of etcd-cluster.
func (c *ClusterClient) List(opts v1.ListOptions) (*ClusterList, error) {
	result := &ClusterList{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource(TPRKind).
		VersionedParams(&opts, kapi.ParameterCodec).
		Do().
		Into(result)
	return result, err
}

// Watch returns a watch.Interface that watches the requested etcd
func (c *ClusterClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Namespace(c.ns).
		Prefix("watch").
		Resource(TPRKind).
		VersionedParams(&opts, kapi.ParameterCodec).
		Watch()
}

// Update a given etcd-cluster.
func (c *ClusterClient) Update(etcd *Cluster) (*Cluster, error) {
	result := &Cluster{}
	err := c.client.Put().
		Namespace(c.ns).
		Resource(TPRKind).
		Name(etcd.Metadata.Name).
		Body(etcd).
		Do().
		Into(result)
	return result, err
}

// Delete takes the name of a etcd-cluster. and removes it from the TPR
func (c *ClusterClient) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource(TPRKind).
		Name(name).
		Body(options).
		Do().
		Error()
}

// Get takes the name of a etcd-cluster and fetches it from the TPR.
func (c *ClusterClient) Get(name string) (*Cluster, error) {
	result := &Cluster{}
	err := c.client.Get().
		Namespace(c.ns).
		Resource(TPRKind).
		Name(name).
		Do().
		Into(result)
	return result, err
}
