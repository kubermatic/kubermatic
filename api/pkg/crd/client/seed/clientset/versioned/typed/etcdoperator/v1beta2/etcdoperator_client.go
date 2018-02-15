package v1beta2

import (
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned/scheme"
	v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type EtcdV1beta2Interface interface {
	RESTClient() rest.Interface
	EtcdClustersGetter
}

// EtcdV1beta2Client is used to interact with features provided by the etcd.database.coreos.com group.
type EtcdV1beta2Client struct {
	restClient rest.Interface
}

func (c *EtcdV1beta2Client) EtcdClusters(namespace string) EtcdClusterInterface {
	return newEtcdClusters(c, namespace)
}

// NewForConfig creates a new EtcdV1beta2Client for the given config.
func NewForConfig(c *rest.Config) (*EtcdV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &EtcdV1beta2Client{client}, nil
}

// NewForConfigOrDie creates a new EtcdV1beta2Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *EtcdV1beta2Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new EtcdV1beta2Client for the given RESTClient.
func New(c rest.Interface) *EtcdV1beta2Client {
	return &EtcdV1beta2Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta2.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *EtcdV1beta2Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
