package versioned

import (
	glog "github.com/golang/glog"
	etcdv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/seed/clientset/versioned/typed/etcd/v1beta2"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	EtcdV1beta2() etcdv1beta2.EtcdV1beta2Interface
	// Deprecated: please explicitly pick a version if possible.
	Etcd() etcdv1beta2.EtcdV1beta2Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	etcdV1beta2 *etcdv1beta2.EtcdV1beta2Client
}

// EtcdV1beta2 retrieves the EtcdV1beta2Client
func (c *Clientset) EtcdV1beta2() etcdv1beta2.EtcdV1beta2Interface {
	return c.etcdV1beta2
}

// Deprecated: Etcd retrieves the default version of EtcdClient.
// Please explicitly pick a version.
func (c *Clientset) Etcd() etcdv1beta2.EtcdV1beta2Interface {
	return c.etcdV1beta2
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.etcdV1beta2, err = etcdv1beta2.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.etcdV1beta2 = etcdv1beta2.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.etcdV1beta2 = etcdv1beta2.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
