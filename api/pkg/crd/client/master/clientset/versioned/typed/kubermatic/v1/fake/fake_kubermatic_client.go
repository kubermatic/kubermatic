package fake

import (
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned/typed/kubermatic/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKubermaticV1 struct {
	*testing.Fake
}

func (c *FakeKubermaticV1) Clusters() v1.ClusterInterface {
	return &FakeClusters{c}
}

func (c *FakeKubermaticV1) UserSSHKeies() v1.UserSSHKeyInterface {
	return &FakeUserSSHKeies{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKubermaticV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
