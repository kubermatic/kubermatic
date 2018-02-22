package cluster

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPendingCreateAddressesSuccessfully(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			SeedDatacenterName: TestDC,
		},
		Address: &kubermaticv1.ClusterAddress{},
	}

	if err := f.controller.ensureAddress(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExternalName := fmt.Sprintf("%s.%s.%s", c.Name, TestDC, TestExternalURL)
	if c.Address.ExternalName != fmt.Sprintf("%s.%s.%s", c.Name, TestDC, TestExternalURL) {
		t.Fatalf("external name is wrong. Expected=%s Got=%s", expectedExternalName, c.Address.ExternalName)
	}

	if c.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, c.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", c.Address.ExternalName, TestExternalPort)
	if c.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, c.Address.URL)
	}
}

func TestPendingCreateAddressesPartially(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			SeedDatacenterName: TestDC,
		},
		Address: &kubermaticv1.ClusterAddress{
			ExternalName: fmt.Sprintf("%s.%s.dev.kubermatic.io", TestClusterName, TestDC),
		},
	}

	if err := f.controller.ensureAddress(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Address.ExternalName != fmt.Sprintf("%s.%s.dev.kubermatic.io", TestClusterName, TestDC) {
		t.Fatalf("external got overwritten")
	}

	if c.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, c.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", c.Address.ExternalName, TestExternalPort)
	if c.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, c.Address.URL)
	}
}

func TestPendingCreateAddressesAlreadyExists(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			SeedDatacenterName: TestDC,
		},
		Address: &kubermaticv1.ClusterAddress{
			ExternalName: "fqpcvnc6v.europe-west3-c.dev.kubermatic.io",
			URL:          "https://fqpcvnc6v.europe-west3-c.dev.kubermatic.io:30004",
			ExternalPort: 30004,
			IP:           "35.198.93.90",
		},
	}

	if err := f.controller.ensureAddress(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Address.ExternalName != "fqpcvnc6v.europe-west3-c.dev.kubermatic.io" || c.Address.URL != "https://fqpcvnc6v.europe-west3-c.dev.kubermatic.io:30004" || c.Address.ExternalPort != 30004 {
		t.Fatalf("address fields were overwritten")
	}
}

func TestLaunchingCreateNamespace(t *testing.T) {
	tests := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		err     error
		ns      *corev1.Namespace
	}{
		{
			name: "successfully created",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: TestDC,
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-henrik1",
				},
			},
		},
		{
			name: "already exists",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Spec: kubermaticv1.ClusterSpec{
					SeedDatacenterName: TestDC,
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-henrik1",
				},
			},
			ns: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cluster-henrik1"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var objects []runtime.Object
			if test.ns != nil {
				objects = append(objects, test.ns)
			}
			f := newTestController(objects, []runtime.Object{}, []runtime.Object{})
			beforeActionCount := len(f.kubeclient.Actions())
			err := f.controller.ensureNamespaceExists(test.cluster)
			if err != nil {
				t.Errorf("failed to create namespace: %v", err)
			}
			if test.ns != nil {
				if len(f.kubeclient.Actions()) != beforeActionCount {
					t.Error("client made call to create namespace although a namespace already existed", f.kubeclient.Actions()[beforeActionCount:])
				}
			} else {
				if len(f.kubeclient.Actions()) != beforeActionCount+1 {
					t.Error("client made more more or less than 1 call to create namespace", f.kubeclient.Actions()[beforeActionCount:])
				}
			}
		})
	}
}
