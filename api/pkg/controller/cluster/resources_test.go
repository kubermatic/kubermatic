package cluster

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

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
				Address: kubermaticv1.ClusterAddress{},
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
				Address: kubermaticv1.ClusterAddress{},
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
			controller := newTestController(objects, []runtime.Object{test.cluster})
			beforeActionCount := len(controller.kubeClient.(*fake.Clientset).Actions())
			_, err := controller.ensureNamespaceExists(test.cluster)
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

func TestConfigMapCreatorsKeepAdditionalData(t *testing.T) {
	cluster := &kubermaticv1.Cluster{}
	cluster.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"10.10.0.0/8"}
	cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.11.0.0/8"}
	dc := &provider.DatacenterMeta{}
	templateData := &resources.TemplateData{
		Cluster:           cluster,
		DC:                dc,
		NodeAccessNetwork: "10.12.0.0/8",
	}

	for _, create := range GetConfigMapCreators() {
		existing := &corev1.ConfigMap{
			Data: map[string]string{"Test": "Data"},
		}
		new, err := create(templateData, existing)
		if err != nil {
			t.Fatalf("Error executing configmap creator: %v", err)
		}

		if val, exists := new.Data["Test"]; !exists || val != "Data" {
			t.Fatalf("Configmap creator for %s removed additional data!", new.Name)
		}
	}
}

func TestSecretV2CreatorsKeepAdditionalData(t *testing.T) {
	cluster := &kubermaticv1.Cluster{}
	cluster.Status.NamespaceName = "test-ns"
	dc := &provider.DatacenterMeta{}
	controller := Controller{}

	keyPair := triple.NewCA("test-ca")

	caKeySecret := &corev1.Secret{}
	caKeySecret.Name = resources.CACertSecretKey
	caKeySecret.Namespace = "test-ns"
	data, err := controller.createRootCAKeySecret
	if err != nil {
		t.Fatalf("Failed to create private key for root ca: %v", err)
	}
	caKeySecret.Data = data

	caCertSecret := &corev1.Secret{}
	caKeySecret.Name = resources.CACertSecretName
	caKeySecret.Namespace = "test-ns"
	data, err := controller.createRootCAKeySecret
	if err != nil {
		t.Fatalf("Failed to create private key for root ca: %v", err)
	}
	caKeySecret.Data = data

	secretIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := secretIndexer.Add(caCertSecret); err != nil {
		t.Fatalf("Error adding secret to indexer: %v", err)
	}
	secretLister := listerscorev1.NewSecretLister(secretIndexer)

	templateData := &resources.TemplateData{
		Cluster:      cluster,
		DC:           dc,
		SecretLister: secretLister,
	}

	for name, create := range GetSecretCreators() {
		existing := &corev1.Secret{
			Data: map[string][]byte{"Test": []byte("val")},
		}
		new, err := create(templateData, existing)
		if err != nil {
			t.Fatalf("Error executing secet creator %s: %v", name, err)
		}

		if val, exists := new.Data["Test"]; !exists || string(val) != "Data" {
			t.Fatalf("Secret creator for %s removed additional data!", new.Name)
		}
	}
}
