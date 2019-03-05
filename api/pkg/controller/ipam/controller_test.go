package ipam

import (
	"strings"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1fakeclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake"
	clusterinformers "sigs.k8s.io/cluster-api/pkg/client/informers_generated/externalversions"
)

func TestFailWhenCIDRIsExhausted(t *testing.T) {
	t.Parallel()

	nets := []Network{buildNet(t, "192.168.0.0/30", "192.168.0.1", "8.8.8.8")}

	mSusi := createMachine("susi")
	mBabsi := createMachine("babsi")
	mJoan := createMachine("joan")

	ctrl, stop := newTestController(nets, mSusi, mBabsi, mJoan)
	defer close(stop)

	err := ctrl.syncMachine(mSusi)
	if err != nil {
		t.Errorf("error in machineAdded handler: %v", err)
	}

	err = ctrl.syncMachine(mBabsi)
	if err != nil {
		t.Errorf("error in machineAdded handler: %v", err)
	}

	err = ctrl.syncMachine(mJoan)
	if err == nil || !strings.Contains(err.Error(), "because no more ips can be allocated from the specified cidrs") {
		t.Error("Expected error for exhausted CIDR range but didnt get it :-(")
	}
}

func assertNetworkEquals(t *testing.T, m *clusterv1alpha1.Machine, ip string, gw string, dns ...string) {
	network, err := getNetworkForMachine(m)
	if err != nil {
		t.Errorf("couldn't get network for machine %s, see: %v", m.Name, err)
	}

	if network.CIDR != ip {
		t.Errorf("Assertion mismatch for machine %s, see: expected cidr '%s' but got '%s'", m.Name, ip, network.CIDR)
	}

	if network.Gateway != gw {
		t.Errorf("Assertion mismatch for machine %s, see: expected gateway '%s' but got '%s'", m.Name, gw, network.Gateway)
	}

	expectedDNSJoined := strings.Join(dns, ",")
	actualDNSJoined := strings.Join(network.DNS.Servers, ",")

	if expectedDNSJoined != actualDNSJoined {
		t.Errorf("Assertion mismatch for machine %s, see: expected dns servers '%s' but got '%s'", m.Name, expectedDNSJoined, actualDNSJoined)
	}
}

func getNetworkForMachine(m *clusterv1alpha1.Machine) (*providerconfig.NetworkConfig, error) {
	cfg, err := providerconfig.GetConfig(m.Spec.ProviderSpec)
	if err != nil {
		return nil, err
	}

	return cfg.Network, nil
}

func newTestController(networks []Network, objects ...runtime.Object) (*Controller, chan struct{}) {
	tweakFunc := func(options *metav1.ListOptions) {
		options.IncludeUninitialized = true
	}

	client := clusterv1alpha1fakeclientset.NewSimpleClientset(objects...)
	factory := clusterinformers.NewFilteredSharedInformerFactory(client, 1*time.Second, metav1.NamespaceAll, tweakFunc)

	controller := NewController(client, factory.Cluster().V1alpha1().Machines(), networks)
	stopCh := make(chan struct{})

	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)

	return controller, stopCh
}
