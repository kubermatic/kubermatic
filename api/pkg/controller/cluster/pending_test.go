package cluster

import (
	"fmt"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPendingCreateAddressesSuccessfully(t *testing.T) {
	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: TestClusterName,
		},
		Spec:    kubermaticv1.ClusterSpec{},
		Address: &kubermaticv1.ClusterAddress{},
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
	controller := newTestController([]runtime.Object{externalService}, []runtime.Object{})

	if err := controller.ensureAddress(c); err != nil {
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
				Spec:    kubermaticv1.ClusterSpec{},
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
				Spec:    kubermaticv1.ClusterSpec{},
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
			controller := newTestController(objects, []runtime.Object{})
			beforeActionCount := len(controller.kubeClient.(*fake.Clientset).Actions())
			err := controller.ensureNamespaceExists(test.cluster)
			if err != nil {
				t.Errorf("failed to create namespace: %v", err)
			}
			if test.ns != nil {
				if len(controller.kubeClient.(*fake.Clientset).Actions()) != beforeActionCount {
					t.Error("client made call to create namespace although a namespace already existed", controller.kubeClient.(*fake.Clientset).Actions()[beforeActionCount:])
				}
			} else {
				if len(controller.kubeClient.(*fake.Clientset).Actions()) != beforeActionCount+1 {
					t.Error("client made more more or less than 1 call to create namespace", controller.kubeClient.(*fake.Clientset).Actions()[beforeActionCount:])
				}
			}
		})
	}
}

func TestEnsureAutomaticMasterUpdate(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		err         error
		wantVersion string
	}{
		{
			name:        "automatic from 1.8.0 to 1.8.10",
			wantVersion: "1.8.10",
			err:         nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Spec: kubermaticv1.ClusterSpec{
					MasterVersion: "1.8.0",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
			},
		},
		{
			name:        "automatic from 1.8.* to 1.8.10",
			wantVersion: "1.8.10",
			err:         nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Spec: kubermaticv1.ClusterSpec{
					MasterVersion: "1.8.3",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
			},
		},
		{
			name:        "no auto update from 1.8.10",
			wantVersion: "1.8.10",
			err:         nil,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Spec: kubermaticv1.ClusterSpec{
					MasterVersion: "1.8.10",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status:  kubermaticv1.ClusterStatus{},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := newTestController([]runtime.Object{}, []runtime.Object{})
			err := controller.ensureAutomaticMasterUpdate(test.cluster)
			if err != nil {
				t.Errorf("failed to ensure automatic update: %v", err)
			}
			if test.cluster.Spec.MasterVersion != test.wantVersion {
				t.Errorf("got invalid version from update. Want: %s Got %s", test.wantVersion, test.cluster.Spec.MasterVersion)
			}
		})
	}
}
