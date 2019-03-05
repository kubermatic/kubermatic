package ipam

import (
	"context"
	"net"
	"testing"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	// We call this in init because even thought it is possible to register the same
	// scheme multiple times it is an unprotected concurrent map access and these tests
	// are very good at making that panic
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		glog.Fatalf("failed to add clusterv1alpha1 scheme to scheme.Scheme: %v", err)
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

func createMachine(name string) *clusterv1alpha1.Machine {
	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   metav1.NamespaceSystem,
			Annotations: map[string]string{"kubermatic/io/initialization": "ipam"},
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
