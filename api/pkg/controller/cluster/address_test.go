package cluster

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Henrik created 2 dns entries for dns-test @ OVH.com. dns-test.kubermatic.io points to:
// - 192.168.1.1
// - 192.168.1.2
// - 2001:16B8:6844:D700:A1B9:D94B:FDC3:1C33
func TestGetExternalIPv4(t *testing.T) {
	ip, err := getExternalIPv4("dns-test.kubermatic.io")
	if err != nil {
		t.Fatal(err)
	}

	if ip != "192.168.1.1" {
		t.Fatalf("expected to get 192.168.1.1. Got: %s", ip)
	}
}

func TestPendingCreateAddressesSuccessfully(t *testing.T) {
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "regular-do1",
			},
		},
		Address: kubermaticv1.ClusterAddress{},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-" + TestClusterName,
		},
	}
	externalService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ApiserverExternalServiceName,
			Namespace: "cluster-" + TestClusterName,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					NodePort: 30000,
				},
			},
		},
	}
	controller := newTestController([]runtime.Object{externalService}, []runtime.Object{c})

	updatedCluster, err := controller.syncAddress(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExternalName := fmt.Sprintf("%s.%s.%s", updatedCluster.Name, TestDC, TestExternalURL)
	if updatedCluster.Address.ExternalName != fmt.Sprintf("%s.%s.%s", updatedCluster.Name, TestDC, TestExternalURL) {
		t.Fatalf("external name is wrong. Expected=%s Got=%s", expectedExternalName, updatedCluster.Address.ExternalName)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", updatedCluster.Address.ExternalName, TestExternalPort)
	if updatedCluster.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, updatedCluster.Address.URL)
	}
}

func TestSeedDNSOverride(t *testing.T) {
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "dns-override-do2",
			},
		},
		Address: kubermaticv1.ClusterAddress{},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-" + TestClusterName,
		},
	}
	externalService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ApiserverExternalServiceName,
			Namespace: "cluster-" + TestClusterName,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					NodePort: 30000,
				},
			},
		},
	}
	controller := newTestController([]runtime.Object{externalService}, []runtime.Object{c})

	updatedCluster, err := controller.syncAddress(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedExternalName := fmt.Sprintf("%s.%s.%s", updatedCluster.Name, "alias-europe-west3-c", TestExternalURL)
	if updatedCluster.Address.ExternalName != expectedExternalName {
		t.Fatalf("external name is wrong. Expected=%s Got=%s", expectedExternalName, updatedCluster.Address.ExternalName)
	}

	expectedURL := fmt.Sprintf("https://%s:%d", updatedCluster.Address.ExternalName, TestExternalPort)
	if updatedCluster.Address.URL != expectedURL {
		t.Fatalf("url is wrong. Expected=%s Got=%s", expectedURL, updatedCluster.Address.URL)
	}
}
