package fake

import (
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/addons/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeAddonsV1alpha1 struct {
	*testing.Fake
}

func (c *FakeAddonsV1alpha1) Addons() v1alpha1.AddonInterface {
	return &FakeAddons{c}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeAddonsV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
