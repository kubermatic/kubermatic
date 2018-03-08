package fake

import (
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/prometheus/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeMonitoringV1 struct {
	*testing.Fake
}

func (c *FakeMonitoringV1) Prometheuses(namespace string) v1.PrometheusInterface {
	return &FakePrometheuses{c, namespace}
}

func (c *FakeMonitoringV1) ServiceMonitors(namespace string) v1.ServiceMonitorInterface {
	return &FakeServiceMonitors{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeMonitoringV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
