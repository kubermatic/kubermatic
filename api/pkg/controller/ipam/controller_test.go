package ipam

import (
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1fakeclientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/fake"
	clusterinformers "sigs.k8s.io/cluster-api/pkg/client/informers_generated/externalversions"
)

type machineTestData struct {
	ip      string
	gw      string
	machine *clusterv1alpha1.Machine
}

// TestIPAMController is a wrapper around all IPAM controller tests
// that setups a custom fake client within the vendor dir and clears
// it up after the tests. This is required because the cluster-api
// clientset has a bug that results in the listers not getting updates
// This issue got fixed in client-go 1.10 (kubernetes/kubernetes#57504)
// but not in client-go 1.9 which they use
//go:generate ./testdata/gen-clusterapi-client.sh
func TestIPAMController(t *testing.T) {
	for name, testCase := range testCases {
		t.Run(name, testCase)
	}
	if out, err := exec.Command("./testdata/cleanup-clusterapi-client.sh").CombinedOutput(); err != nil {
		t.Fatalf("Failed to clean up clientset for testing: err=%v, out=\n%v", err, string(out))
	}
}

var testCases = map[string]func(*testing.T){
	"TestSingleCIDRAllocation": func(t *testing.T) {
		t.Parallel()

		nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

		m := createMachine("susi")
		ctrl, stop := newTestController(nets, m)
		defer close(stop)

		err := ctrl.syncMachine(m)
		if err != nil {
			t.Errorf("error in machineAdded handler: %v", err)
		}

		m2, err := ctrl.client.ClusterV1alpha1().Machines(m.Namespace).Get("susi", metav1.GetOptions{})
		if err != nil {
			t.Errorf("couldn't retrieve updated machine, see: %v", err)
		}

		assertNetworkEquals(t, m2, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")
	},

	"TestMultipleCIDRAllocation": func(t *testing.T) {
		t.Parallel()

		nets := []Network{
			buildNet(t, "192.168.0.0/30", "192.168.0.1", "8.8.8.8"),
			buildNet(t, "10.0.0.0/24", "10.0.0.1", "8.8.8.8"),
		}

		machines := []machineTestData{
			{"192.168.0.2/30", "192.168.0.1", createMachine("susi")},
			{"192.168.0.3/30", "192.168.0.1", createMachine("babsi")},
			{"10.0.0.2/24", "10.0.0.1", createMachine("joan")},
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

			m2, err := ctrl.client.ClusterV1alpha1().Machines(tuple.machine.Namespace).Get(tuple.machine.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("couldn't retrieve updated machine, see: %v", err)
			}

			assertNetworkEquals(t, m2, tuple.ip, tuple.gw, "8.8.8.8")
		}
	},

	"TestReuseReleasedIP": func(t *testing.T) {
		t.Parallel()

		nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

		mSusi := createMachine("susi")
		mBabsi := createMachine("babsi")

		ctrl, stop := newTestController(nets, mSusi, mBabsi)
		defer close(stop)

		err := ctrl.syncMachine(mSusi)
		if err != nil {
			t.Errorf("error in machineAdded handler: %v", err)
		}

		mSusi2, err := ctrl.client.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Get("susi", metav1.GetOptions{})
		if err != nil {
			t.Errorf("couldn't retrieve updated machine, see: %v", err)
		}

		assertNetworkEquals(t, mSusi2, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")

		err = ctrl.client.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Delete("susi", &metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("couldn't retrieve updated machine, see: %v", err)
		}
		err = wait.Poll(5*time.Millisecond, 5*time.Second, func() (bool, error) {
			_, err = ctrl.client.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Get("susi", metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("failed waiting until lister received delete event on machine 'susi': %v", err)
		}

		err = ctrl.syncMachine(mBabsi)
		if err != nil {
			t.Errorf("error in machineAdded handler: %v", err)
		}

		mBabsi2, err := ctrl.client.ClusterV1alpha1().Machines(metav1.NamespaceSystem).Get("babsi", metav1.GetOptions{})
		if err != nil {
			t.Errorf("couldn't retrieve updated machine, see: %v", err)
		}

		assertNetworkEquals(t, mBabsi2, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")
	},

	"TestFailWhenCIDRIsExhausted": func(t *testing.T) {
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
	},
}

func createMachine(name string) *clusterv1alpha1.Machine {
	machine := &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceSystem,
			Initializers: &metav1.Initializers{
				Pending: []metav1.Initializer{{Name: initializerName}},
			},
		},
		Spec: clusterv1alpha1.MachineSpec{
			ProviderConfig: clusterv1alpha1.ProviderConfig{
				Value: &runtime.RawExtension{Raw: []byte(`{}`)},
			},
		},
	}

	return machine
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
	cfg, err := providerconfig.GetConfig(m.Spec.ProviderConfig)
	if err != nil {
		return nil, err
	}

	return cfg.Network, nil
}
