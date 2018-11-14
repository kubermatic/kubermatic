package v1alpha1

import (
	"github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

type ClusterV1alpha1Interface interface {
	RESTClient() rest.Interface
	ClustersGetter
	MachinesGetter
	MachineClassesGetter
	MachineDeploymentsGetter
	MachineSetsGetter
}

// ClusterV1alpha1Client is used to interact with features provided by the cluster.k8s.io group.
type ClusterV1alpha1Client struct {
	restClient rest.Interface
}

func (c *ClusterV1alpha1Client) Clusters(namespace string) ClusterInterface {
	return newClusters(c, namespace)
}

func (c *ClusterV1alpha1Client) Machines(namespace string) MachineInterface {
	return newMachines(c, namespace)
}

func (c *ClusterV1alpha1Client) MachineClasses(namespace string) MachineClassInterface {
	return newMachineClasses(c, namespace)
}

func (c *ClusterV1alpha1Client) MachineDeployments(namespace string) MachineDeploymentInterface {
	return newMachineDeployments(c, namespace)
}

func (c *ClusterV1alpha1Client) MachineSets(namespace string) MachineSetInterface {
	return newMachineSets(c, namespace)
}

// NewForConfig creates a new ClusterV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*ClusterV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ClusterV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new ClusterV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *ClusterV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ClusterV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *ClusterV1alpha1Client {
	return &ClusterV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
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
func (c *ClusterV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
