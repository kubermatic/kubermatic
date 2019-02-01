package cluster

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	cluster.Spec.Version = *semver.NewSemverOrDie("v1.11.1")
	dc := &provider.DatacenterMeta{}
	templateData := resources.NewTemplateData(cluster, dc, "", nil, nil, nil, "", "", "10.12.0.0/8", resource.Quantity{}, "", "", false, false, "", nil, "", "", "", false)

	for _, create := range GetConfigMapCreators(templateData) {
		existing := &corev1.ConfigMap{
			Data: map[string]string{"Test": "Data"},
		}
		new, err := create(existing)
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
	testNamespace := "test-ns"
	clusterIP := "1.2.3.4"
	cluster.Status.NamespaceName = testNamespace
	cluster.Address.IP = clusterIP
	cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	cluster.Spec.MachineNetworks = []kubermaticv1.MachineNetworkingConfig{{CIDR: "10.11.11.0/24", Gateway: "10.11.11.1", DNSServers: []string{"10.11.11.2"}}}
	dc := &provider.DatacenterMeta{}

	keyPair, err := triple.NewCA("test-ca")
	if err != nil {
		t.Fatalf("Failed to generate test root ca: %v", err)
	}
	caSecret := &corev1.Secret{}
	caSecret.Name = resources.CASecretName
	caSecret.Namespace = testNamespace
	caSecret.Data = map[string][]byte{
		resources.CACertSecretKey: certutil.EncodeCertPEM(keyPair.Cert),
		resources.CAKeySecretKey:  certutil.EncodePrivateKeyPEM(keyPair.Key),
	}

	frontProxyCASecret := &corev1.Secret{}
	frontProxyCASecret.Name = resources.FrontProxyCASecretName
	frontProxyCASecret.Namespace = testNamespace
	frontProxyCASecret.Data = map[string][]byte{
		resources.CACertSecretKey: certutil.EncodeCertPEM(keyPair.Cert),
		resources.CAKeySecretKey:  certutil.EncodePrivateKeyPEM(keyPair.Key),
	}

	openVPNCAcert, openVPNCAkey, err := certificates.GetECDSACACertAndKey()
	if err != nil {
		t.Fatalf("Failed to generate test openVPN ca: %v", err)
	}
	openVPNCASecret := &corev1.Secret{}
	openVPNCASecret.Name = resources.OpenVPNCASecretName
	openVPNCASecret.Namespace = testNamespace
	openVPNCASecret.Data = map[string][]byte{
		resources.OpenVPNCACertKey: openVPNCAcert,
		resources.OpenVPNCAKeyKey:  openVPNCAkey,
	}

	apiserverExternalService := &corev1.Service{}
	apiserverExternalService.Name = resources.ApiserverExternalServiceName
	apiserverExternalService.Namespace = testNamespace
	apiserverExternalService.Spec.ClusterIP = clusterIP
	apiserverExternalService.Spec.Ports = []corev1.ServicePort{
		{
			Name:     "external",
			NodePort: 30443,
		},
	}

	apiserverService := &corev1.Service{}
	apiserverService.Name = resources.ApiserverInternalServiceName
	apiserverService.Namespace = testNamespace
	apiserverService.Spec.ClusterIP = clusterIP

	secretIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := secretIndexer.Add(caSecret); err != nil {
		t.Fatalf("Error adding secret to indexer: %v", err)
	}
	if err := secretIndexer.Add(frontProxyCASecret); err != nil {
		t.Fatalf("Error adding secret to indexer: %v", err)
	}
	if err := secretIndexer.Add(openVPNCASecret); err != nil {
		t.Fatalf("Error adding openVPN ca secret to indexer: %v", err)
	}
	secretLister := listerscorev1.NewSecretLister(secretIndexer)

	serviceIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := serviceIndexer.Add(apiserverExternalService); err != nil {
		t.Fatalf("Error adding service to indexer: %v", err)
	}
	if err := serviceIndexer.Add(apiserverService); err != nil {
		t.Fatalf("Error adding service to indexer: %v", err)
	}
	serviceLister := listerscorev1.NewServiceLister(serviceIndexer)

	file, err := ioutil.TempFile("", "caBundle.pem")
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Write(certutil.EncodeCertPEM(keyPair.Cert))
	if err != nil {
		log.Fatal(err)
	}

	defer removeCloseFile(file)

	templateData := resources.NewTemplateData(cluster, dc, "", secretLister, nil, serviceLister, "", "", "", resource.Quantity{}, "", "", false, false, "", nil, file.Name(), "", "", false)

	for _, op := range GetSecretCreatorOperations(cluster, []byte{}, true) {
		existing := &corev1.Secret{
			Data: map[string][]byte{"Test": []byte("Data")},
		}
		new, err := op.create(templateData, existing)
		if err != nil {
			t.Fatalf("Error executing secet creator %s: %v", op.name, err)
		}

		if val, exists := new.Data["Test"]; !exists || string(val) != "Data" {
			t.Fatalf("Secret creator for %s removed additional data!", new.Name)
		}
	}
}

func removeCloseFile(file *os.File) {
	err := os.Remove(file.Name())
	if err != nil {
		log.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		glog.Fatal(err)
	}
}
