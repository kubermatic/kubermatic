package fake

import (
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/clientset/versioned/typed/cluster/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeClusterV1alpha1 struct {
	*testing.Fake
}

func (c *FakeClusterV1alpha1) Clusters(namespace string) v1alpha1.ClusterInterface {
	return &FakeClusters{c, namespace}
}

func (c *FakeClusterV1alpha1) Machines(namespace string) v1alpha1.MachineInterface {
	return &FakeMachines{c, namespace}
}

func (c *FakeClusterV1alpha1) MachineClasses(namespace string) v1alpha1.MachineClassInterface {
	return &FakeMachineClasses{c, namespace}
}

func (c *FakeClusterV1alpha1) MachineDeployments(namespace string) v1alpha1.MachineDeploymentInterface {
	return &FakeMachineDeployments{c, namespace}
}

func (c *FakeClusterV1alpha1) MachineSets(namespace string) v1alpha1.MachineSetInterface {
	return &FakeMachineSets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeClusterV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
