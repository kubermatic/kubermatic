package ipam

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"go.uber.org/zap"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalw("failed to add clusterv1alpha1 scheme to scheme.Scheme", zap.Error(err))
	}
}

func TestSingleCIDRAllocation(t *testing.T) {
	t.Parallel()

	nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

	m := createMachine("Malcolm")
	r := newTestReconciler(nets, m)

	if err := r.reconcile(context.Background(), m); err != nil {
		t.Fatalf("failed to reconcile machine: %v", err)
	}

	resultMachine := &clusterv1alpha1.Machine{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: "Malcolm"}, resultMachine); err != nil {
		t.Fatalf("failed to get machine after reconciling: %v", err)
	}

	assertNetworkEquals(t, resultMachine, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")
}

func TestMultipleCIDRAllocation(t *testing.T) {
	t.Parallel()

	nets := []Network{
		buildNet(t, "192.168.0.0/30", "192.168.0.1", "8.8.8.8"),
		buildNet(t, "10.0.0.0/24", "10.0.0.1", "8.8.8.8"),
	}

	machines := []machineTestData{
		{"192.168.0.2/30", "192.168.0.1", createMachine("Jayne")},
		{"192.168.0.3/30", "192.168.0.1", createMachine("Kaylee")},
		{"10.0.0.2/24", "10.0.0.1", createMachine("River")},
	}

	machineObjects := []runtime.Object{}
	for _, m := range machines {
		machineObjects = append(machineObjects, m.machine)
	}

	r := newTestReconciler(nets, machineObjects...)
	for _, tuple := range machines {
		if err := r.reconcile(context.Background(), tuple.machine); err != nil {
			t.Errorf("failed to sync machine %q: %v", tuple.machine.Name, err)
		}
		reconciledMachine := &clusterv1alpha1.Machine{}
		if err := r.Get(context.Background(), types.NamespacedName{Name: tuple.machine.Name}, reconciledMachine); err != nil {
			t.Errorf("failed to get machine %q after reconcile: %v", tuple.machine.Name, err)
		}
		assertNetworkEquals(t, reconciledMachine, tuple.ip, tuple.gw, "8.8.8.8")
	}

}

func TestReuseReleasedIP(t *testing.T) {
	t.Parallel()

	nets := []Network{buildNet(t, "192.168.0.0/16", "192.168.0.1", "8.8.8.8")}

	mHoban := createMachine("Hoban")
	mShepherd := createMachine("Shepherd")

	r := newTestReconciler(nets, mHoban, mShepherd)
	if err := r.reconcile(context.Background(), mHoban); err != nil {
		t.Fatalf("failed to sync machine: %v", err)
	}

	updatedHoban := &clusterv1alpha1.Machine{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: mHoban.Name}, updatedHoban); err != nil {
		t.Fatalf("failed to get machine %q after reconcile: %v", mHoban.Name, err)
	}

	assertNetworkEquals(t, updatedHoban, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")

	if err := r.Delete(context.Background(), updatedHoban); err != nil {
		t.Fatalf("failed to delete machine: %v", err)
	}

	if err := r.reconcile(context.Background(), mShepherd); err != nil {
		t.Fatalf("failed to sync machine: %v", err)
	}

	updatedShepherd := &clusterv1alpha1.Machine{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: mShepherd.Name}, updatedShepherd); err != nil {
		t.Fatalf("failed to get machine %q  after reconcile: %v", mShepherd.Name, err)
	}

	assertNetworkEquals(t, updatedShepherd, "192.168.0.2/16", "192.168.0.1", "8.8.8.8")
}

func TestFailWhenCIDRIsExhausted(t *testing.T) {
	t.Parallel()

	nets := []Network{buildNet(t, "192.168.0.0/30", "192.168.0.1", "8.8.8.8")}

	mSimon := createMachine("Simon")
	mZoe := createMachine("Zoe")
	mInara := createMachine("Inara")

	r := newTestReconciler(nets, mSimon, mZoe, mInara)
	if err := r.reconcile(context.Background(), mSimon); err != nil {
		t.Fatalf("failed to reconcile machine %q: %v", mSimon.Name, err)
	}
	if err := r.reconcile(context.Background(), mZoe); err != nil {
		t.Fatalf("failed to reconcile machine %q: %v", mZoe.Name, err)
	}
	if err := r.reconcile(context.Background(), mInara); err == nil || err.Error() != "cidr exhausted" {
		t.Fatalf("Expected err to be 'cidr exhausted' but was %v", err)
	}
}

func createMachine(name string) *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceSystem,
			Annotations: map[string]string{annotationMachineUninitialized: annotationValue},
		},
		Spec: clusterv1alpha1.MachineSpec{
			ProviderSpec: clusterv1alpha1.ProviderSpec{
				Value: &runtime.RawExtension{Raw: []byte(`{}`)},
			},
		},
	}
}

func newTestReconciler(networks []Network, objects ...runtime.Object) *reconciler {
	client := fakectrlruntimeclient.NewFakeClient(objects...)
	return &reconciler{Client: client, cidrRanges: networks}
}

type machineTestData struct {
	ip      string
	gw      string
	machine *clusterv1alpha1.Machine
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
		t.Fatalf("couldn't get network for machine %s, see: %v", m.Name, err)
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

	if cfg.Network == nil {
		return nil, fmt.Errorf("machine %q has no network config", m.Name)
	}

	return cfg.Network, nil
}
