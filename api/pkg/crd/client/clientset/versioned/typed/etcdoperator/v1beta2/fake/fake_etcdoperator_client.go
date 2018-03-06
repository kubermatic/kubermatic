package fake

import (
	v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/etcdoperator/v1beta2"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeEtcdV1beta2 struct {
	*testing.Fake
}

func (c *FakeEtcdV1beta2) EtcdClusters(namespace string) v1beta2.EtcdClusterInterface {
	return &FakeEtcdClusters{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeEtcdV1beta2) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
