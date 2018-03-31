package fake

import (
	clientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	etcdv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/etcdoperator/v1beta2"
	fakeetcdv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/etcdoperator/v1beta2/fake"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	fakekubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1/fake"
	monitoringv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/prometheus/v1"
	fakemonitoringv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/prometheus/v1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o))
	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr, &fakediscovery.FakeDiscovery{Fake: &fakePtr}}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// EtcdV1beta2 retrieves the EtcdV1beta2Client
func (c *Clientset) EtcdV1beta2() etcdv1beta2.EtcdV1beta2Interface {
	return &fakeetcdv1beta2.FakeEtcdV1beta2{Fake: &c.Fake}
}

// Etcd retrieves the EtcdV1beta2Client
func (c *Clientset) Etcd() etcdv1beta2.EtcdV1beta2Interface {
	return &fakeetcdv1beta2.FakeEtcdV1beta2{Fake: &c.Fake}
}

// KubermaticV1 retrieves the KubermaticV1Client
func (c *Clientset) KubermaticV1() kubermaticv1.KubermaticV1Interface {
	return &fakekubermaticv1.FakeKubermaticV1{Fake: &c.Fake}
}

// Kubermatic retrieves the KubermaticV1Client
func (c *Clientset) Kubermatic() kubermaticv1.KubermaticV1Interface {
	return &fakekubermaticv1.FakeKubermaticV1{Fake: &c.Fake}
}

// MonitoringV1 retrieves the MonitoringV1Client
func (c *Clientset) MonitoringV1() monitoringv1.MonitoringV1Interface {
	return &fakemonitoringv1.FakeMonitoringV1{Fake: &c.Fake}
}

// Monitoring retrieves the MonitoringV1Client
func (c *Clientset) Monitoring() monitoringv1.MonitoringV1Interface {
	return &fakemonitoringv1.FakeMonitoringV1{Fake: &c.Fake}
}
