package ipam

import (
	"net"
	"strings"
	"testing"
	"time"

	machinefake "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned/fake"
	machineinformers "github.com/kubermatic/machine-controller/pkg/client/informers/externalversions"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type machineTestData struct {
	ip      string
	gw      string
	machine *machinev1alpha1.Machine
}

func TestSingleCIDRAllocation(t *testing.T) {
	nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

	m := createMachine("susi")
	ctrl, stop := newTestController(nets, m)
	defer close(stop)

	err := ctrl.syncMachine(m)
	if err != nil {
		t.Errorf("error in machineAdded handler: %v", err)
	}

	m2, err := ctrl.client.Machine().Machines().Get("susi", metav1.GetOptions{})
	if err != nil {
		t.Errorf("couldn't retrieve updated machine, see: %v", err)
	}

	assertNetworkEquals(t, m2, "192.168.0.2", "192.168.0.1", "8.8.8.8")
}

func TestMultipleCIDRAllocation(t *testing.T) {
	nets := []Network{
		buildNet(t, "192.168.0.0/30", "192.168.0.1", "8.8.8.8"),
		buildNet(t, "10.0.0.0/24", "10.0.0.1", "8.8.8.8"),
	}

	machines := []machineTestData{
		{"192.168.0.2", "192.168.0.1", createMachine("susi")},
		{"192.168.0.3", "192.168.0.1", createMachine("babsi")},
		{"10.0.0.2", "10.0.0.1", createMachine("joan")},
	}

	machineValues := make([]runtime.Object, 0, len(machines))
	for _, m := range machines {
		machineValues = append(machineValues, m.machine)
	}

	ctrl, stop := newTestController(nets, machineValues...)
	defer close(stop)

	for _, tuple := range machines {
		err := ctrl.syncMachine(tuple.machine)
		if err != nil {
			t.Errorf("error in machineAdded handler: %v", err)
		}

		m2, err := ctrl.client.Machine().Machines().Get(tuple.machine.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("couldn't retrieve updated machine, see: %v", err)
		}

		assertNetworkEquals(t, m2, tuple.ip, tuple.gw, "8.8.8.8")
	}
}

func TestReuseReleasedIP(t *testing.T) {
	nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

	mSusi := createMachine("susi")
	mBabsi := createMachine("babsi")

	ctrl, stop := newTestController(nets, mSusi, mBabsi)
	defer close(stop)

	err := ctrl.syncMachine(mSusi)
	if err != nil {
		t.Errorf("error in machineAdded handler: %v", err)
	}

	mSusi2, err := ctrl.client.Machine().Machines().Get("susi", metav1.GetOptions{})
	if err != nil {
		t.Errorf("couldn't retrieve updated machine, see: %v", err)
	}

	assertNetworkEquals(t, mSusi2, "192.168.0.2", "192.168.0.1", "8.8.8.8")

	now := metav1.NewTime(time.Now())
	mSusi2.DeletionTimestamp = &now

	err = ctrl.syncMachine(mSusi2)
	if err != nil {
		t.Errorf("couldn't delete machine, see: %v", err)
	}

	err = ctrl.syncMachine(mBabsi)
	if err != nil {
		t.Errorf("error in machineAdded handler: %v", err)
	}

	mBabsi2, err := ctrl.client.Machine().Machines().Get("babsi", metav1.GetOptions{})
	if err != nil {
		t.Errorf("couldn't retrieve updated machine, see: %v", err)
	}

	assertNetworkEquals(t, mBabsi2, "192.168.0.2", "192.168.0.1", "8.8.8.8")
}

func TestFailWhenCIDRIsExhausted(t *testing.T) {
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

func createMachine(name string) *machinev1alpha1.Machine {
	machine := &machinev1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Initializers: &metav1.Initializers{
				Pending: []metav1.Initializer{{Name: initializerName}},
			},
		},
		Spec: machinev1alpha1.MachineSpec{
			ProviderConfig: runtime.RawExtension{Raw: []byte{'{', '}'}},
		},
	}

	return machine
}

func newTestController(networks []Network, objects ...runtime.Object) (*Controller, chan struct{}) {
	tweakFunc := func(options *metav1.ListOptions) {
		options.IncludeUninitialized = true
	}

	client := machinefake.NewSimpleClientset(objects...)
	factory := machineinformers.NewFilteredSharedInformerFactory(client, 1*time.Second, metav1.NamespaceAll, tweakFunc)
	informer := factory.Machine().V1alpha1().Machines()

	controller := NewController(client, informer, networks)
	stopCh := make(chan struct{})

	factory.Start(stopCh)

	return controller, stopCh
}

func buildNet(t *testing.T, cidr string, gw string, dnsServers ...string) Network {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("error in network config of test, couldnt parse %s as cidr", cidr)
	}

	dnsIps := make([]net.IP, len(dnsServers))
	for i, d := range dnsServers {
		dnsIps[i] = net.ParseIP(d)
	}

	return Network{
		IP:         ip,
		IPNet:      *ipnet,
		Gateway:    net.ParseIP(gw),
		DNSServers: dnsIps,
	}
}

func assertNetworkEquals(t *testing.T, m *machinev1alpha1.Machine, ip string, gw string, dns ...string) {
	net, err := getNetworkForMachine(m, t)
	if err != nil {
		t.Errorf("couldn't get network for machine %s, see: %v", m.Name, err)
	}

	if net.CIDR != ip {
		t.Errorf("Assertion mismatch for machine %s, see: expected cidr '%s' but got '%s'", m.Name, ip, net.CIDR)
	}

	if net.Gateway != gw {
		t.Errorf("Assertion mismatch for machine %s, see: expected gateway '%s' but got '%s'", m.Name, gw, net.Gateway)
	}

	expectedDNSJoined := strings.Join(dns, ",")
	actualDNSJoined := strings.Join(net.DNS.Servers, ",")

	if expectedDNSJoined != actualDNSJoined {
		t.Errorf("Assertion mismatch for machine %s, see: expected dns servers '%s' but got '%s'", m.Name, expectedDNSJoined, actualDNSJoined)
	}
}

func getNetworkForMachine(m *machinev1alpha1.Machine, t *testing.T) (*providerconfig.NetworkConfig, error) {
	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}

	return cfg.Network, nil
}
