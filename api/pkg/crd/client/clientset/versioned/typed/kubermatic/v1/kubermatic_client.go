package v1

import (
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/scheme"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type KubermaticV1Interface interface {
	RESTClient() rest.Interface
	AddonsGetter
	ClustersGetter
	ProjectsGetter
	UsersGetter
	UserProjectBindingsGetter
	UserSSHKeiesGetter
}

// KubermaticV1Client is used to interact with features provided by the kubermatic.k8s.io group.
type KubermaticV1Client struct {
	restClient rest.Interface
}

func (c *KubermaticV1Client) Addons(namespace string) AddonInterface {
	return newAddons(c, namespace)
}

func (c *KubermaticV1Client) Clusters() ClusterInterface {
	return newClusters(c)
}

func (c *KubermaticV1Client) Projects() ProjectInterface {
	return newProjects(c)
}

func (c *KubermaticV1Client) Users() UserInterface {
	return newUsers(c)
}

func (c *KubermaticV1Client) UserProjectBindings() UserProjectBindingInterface {
	return newUserProjectBindings(c)
}

func (c *KubermaticV1Client) UserSSHKeies() UserSSHKeyInterface {
	return newUserSSHKeies(c)
}

// NewForConfig creates a new KubermaticV1Client for the given config.
func NewForConfig(c *rest.Config) (*KubermaticV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &KubermaticV1Client{client}, nil
}

// NewForConfigOrDie creates a new KubermaticV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *KubermaticV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new KubermaticV1Client for the given RESTClient.
func New(c rest.Interface) *KubermaticV1Client {
	return &KubermaticV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
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
func (c *KubermaticV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
