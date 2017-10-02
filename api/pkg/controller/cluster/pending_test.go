package cluster

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPendingCreateAddressesSuccessfully(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
		},
		Address: &kubermaticv1.ClusterAddress{},
	}

	changedC, err := f.controller.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExternalName := fmt.Sprintf("%s.%s.%s", c.Name, TestDC, TestExternalURL)
	if changedC.Address.ExternalName != fmt.Sprintf("%s.%s.%s", c.Name, TestDC, TestExternalURL) {
		t.Fatalf("external name is wrong. Expected=%s Got=%s", expectedExternalName, changedC.Address.ExternalName)
	}

	if changedC.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, changedC.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", changedC.Address.ExternalName, TestExternalPort)
	if changedC.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, changedC.Address.URL)
	}
}

func TestPendingCreateAddressesPartially(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
		},
		Address: &kubermaticv1.ClusterAddress{
			ExternalName: "foo.bar",
		},
	}

	changedC, err := f.controller.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC.Address.ExternalName != "foo.bar" {
		t.Fatalf("external got overwritten")
	}

	if changedC.Address.ExternalPort != TestExternalPort {
		t.Fatalf("external port is wrong. Expected=%d Got=%d", TestExternalPort, changedC.Address.ExternalPort)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", changedC.Address.ExternalName, TestExternalPort)
	if changedC.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, changedC.Address.URL)
	}
}

func TestPendingCreateAddressesAlreadyExists(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
		},
		Address: &kubermaticv1.ClusterAddress{
			ExternalName: "foo.bar",
			URL:          "https://foo.bar:8443",
			ExternalPort: 8443,
		},
	}

	changedC, err := f.controller.pendingCreateAddresses(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC != nil {
		t.Fatalf("returned cluster pointer to trigger update instead of nil")
	}

	if c.Address.ExternalName != "foo.bar" || c.Address.URL != "https://foo.bar:8443" || c.Address.ExternalPort != 8443 {
		t.Fatalf("address fields were overwritten")
	}
}

func TestLaunchingCreateNamespace(t *testing.T) {
	tests := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		err     error
		ns      *v1.Namespace
	}{
		{
			name: "successfully created",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-henrik1",
				},
			},
			ns: &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "cluster-henrik1"}},
		},
		{
			name: "already exists",
			err:  nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-henrik1",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objects := []runtime.Object{}
			if test.ns != nil {
				objects = append(objects, test.ns)
			}
			f := newTestController(objects, []runtime.Object{}, []runtime.Object{})
			beforeActionCount := len(f.kubeclient.Actions())
			err := f.controller.launchingCreateNamespace(test.cluster)
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
