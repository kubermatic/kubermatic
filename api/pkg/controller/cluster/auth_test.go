package cluster

import (
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
)

func TestCreateServiceAccountKey(t *testing.T) {
	b, err := createServiceAccountKey()

	if err != nil {
		t.Fatal("Unexpected error: ", err)
	}

	if len(b) == 0 {
		t.Fatalf("Expected byte buffer to have length > 0, got %d", len(b))
	}

	s := string(b)
	header := "RSA PRIVATE KEY"
	if !strings.Contains(s, header) {
		t.Fatalf("Expected retured byte buffer to contain %s", header)
	}
}

func TestSSHKeyCert(t *testing.T) {
	k, err := createSSHKey()

	if err != nil {
		t.Fatal("Unexpected error: ", err)
	}

	if len(k.PrivateKey) == 0 {
		t.Fatalf("Expected byte buffer in PrivateKey to have length > 0, got %d", len(k.PrivateKey))
	}

	if len(k.PublicKey) == 0 {
		t.Fatalf("Expected byte buffer in PublicKey to have length > 0, got %d", len(k.PublicKey))
	}

	ks := string(k.PrivateKey)
	header := "RSA PRIVATE KEY"
	if !strings.Contains(ks, header) {
		t.Fatalf("Expected retured k.Key byte buffer to contain %s", header)
	}

	cs := string(k.PublicKey)
	cHeader := "ssh-rsa"
	if !strings.Contains(cs, cHeader) {
		t.Fatalf("Expected retured k.Key byte buffer to contain %s", cHeader)
	}
}

func TestPendingCreateApiserverSSHKeysSuccessfully(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			ApiserverSSHKey: kubermaticv1.SecretRSAKeys{},
		},
	}

	c, err := f.controller.pendingCreateRootCA(c)
	if err != nil {
		t.Fatal(err)
	}

	c, err = f.controller.pendingCreateApiserverSSHKeys(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Status.ApiserverSSHKey.PrivateKey == nil {
		t.Fatalf("apiserver ssh private key was not generated")
	}
	if c.Status.ApiserverSSHKey.PublicKey == nil {
		t.Fatalf("apiserver ssh public key was not generated")
	}

}

func TestPendingCreateApiserverSSHKeysAlreadyExist(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			ApiserverSSHKey: kubermaticv1.SecretRSAKeys{
				PublicKey:  []byte("PUB_KEY"),
				PrivateKey: []byte("PRIV_KEY"),
			},
		},
	}

	c, err := f.controller.pendingCreateRootCA(c)
	if err != nil {
		t.Fatal(err)
	}

	changedC, err := f.controller.pendingCreateApiserverSSHKeys(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC != nil {
		t.Fatalf("returned cluster pointer to trigger update instead of nil")
	}

	if string(c.Status.ApiserverSSHKey.PublicKey) != "PUB_KEY" || string(c.Status.ApiserverSSHKey.PrivateKey) != "PRIV_KEY" {
		t.Fatalf("apiserver ssh key was overwritten")
	}

}

func TestPendingCreateCertificates(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	tests := []struct {
		name               string
		cluster            *kubermaticv1.Cluster
		err                error
		createRootCA       bool
		clusterReturnIsNil bool
	}{
		{
			name:               "successful both",
			err:                nil,
			createRootCA:       true,
			clusterReturnIsNil: false,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:               "successful partial apiserver",
			err:                nil,
			createRootCA:       true,
			clusterReturnIsNil: false,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
				},
			},
		},
		{
			name:               "successful partial kubelet",
			err:                nil,
			createRootCA:       true,
			clusterReturnIsNil: false,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:               "root-ca is missing",
			err:                errors.New("failed to parse root-ca cert: data does not contain any valid RSA or ECDSA certificates"),
			createRootCA:       false,
			clusterReturnIsNil: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:               "external name is missing",
			err:                errors.New("external name is undefined"),
			clusterReturnIsNil: true,
			createRootCA:       true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:               "already exists",
			err:                nil,
			clusterReturnIsNil: true,
			createRootCA:       true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.SecretKeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
					KubeletCert:   kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			if test.createRootCA {
				test.cluster, err = f.controller.pendingCreateRootCA(test.cluster)
				if err != nil {
					t.Errorf("could not create root ca")
				}
			}
			c, err := f.controller.pendingCreateCertificates(test.cluster)
			if !reflect.DeepEqual(test.err, err) {
				t.Errorf("error was %q instead of %q", err, test.err)
			}
			if test.clusterReturnIsNil && c != nil {
				t.Error("cluster was not nil")
			}
			if !test.clusterReturnIsNil && test.err == nil {
				if c.Status.ApiserverCert.Cert == nil {
					t.Error("apiserver cert is nil")
				}
				if c.Status.ApiserverCert.Key == nil {
					t.Error("apiserver cert key is nil")
				}
				if c.Status.KubeletCert.Cert == nil {
					t.Error("kubelet cert is nil")
				}
				if c.Status.KubeletCert.Key == nil {
					t.Error("kubelet cert key is nil")
				}
			}

		})
	}
}

func TestPendingCreateRootCASuccessfully(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			RootCA: kubermaticv1.SecretKeyCert{},
		},
	}

	c, err := f.controller.pendingCreateRootCA(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Status.RootCA.Key == nil {
		t.Fatalf("root ca key is nil")
	}

	if c.Status.RootCA.Cert == nil {
		t.Fatalf("root ca cert is nil")
	}
}

func TestPendingCreateRootCAAlreadyExist(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Status: kubermaticv1.ClusterStatus{
			RootCA: kubermaticv1.SecretKeyCert{
				Cert: kubermaticv1.Bytes([]byte("CERT")),
				Key:  kubermaticv1.Bytes([]byte("KEY")),
			},
		},
	}

	changedC, err := f.controller.pendingCreateRootCA(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC != nil {
		t.Fatalf("returned cluster pointer to trigger update instead of nil")
	}

	if string(c.Status.RootCA.Key) != "KEY" || string(c.Status.RootCA.Cert) != "CERT" {
		t.Fatalf("root ca was overwritten")
	}
}

func TestPendingCreateTokensSuccessfully(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Address: &kubermaticv1.ClusterAddress{},
	}

	changedC, err := f.controller.pendingCreateTokens(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC.Address.KubeletToken == "" {
		t.Fatalf("kubelet token is empty")
	}

	if changedC.Address.AdminToken == "" {
		t.Fatalf("admin token is empty")
	}
}

func TestPendingCreateTokensAlreadyExists(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	c := &kubermaticv1.Cluster{
		Address: &kubermaticv1.ClusterAddress{
			KubeletToken: "KubeletToken",
			AdminToken:   "AdminToken",
		},
	}

	changedC, err := f.controller.pendingCreateTokens(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if changedC != nil {
		t.Fatalf("returned cluster pointer to trigger update instead of nil")
	}

	if c.Address.KubeletToken != "KubeletToken" || c.Address.AdminToken != "AdminToken" {
		t.Fatalf("tokens were overwritten")
	}
}

func TestPendingCreateServiceAccountKey(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	tests := []struct {
		name               string
		cluster            *kubermaticv1.Cluster
		err                error
		clusterReturnIsNil bool
	}{
		{
			name:               "successful",
			err:                nil,
			clusterReturnIsNil: false,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					ServiceAccountKey: nil,
				},
			},
		},
		{
			name:               "already exists",
			err:                nil,
			clusterReturnIsNil: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					ServiceAccountKey: []byte("foo"),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			c, err := f.controller.pendingCreateServiceAccountKey(test.cluster)
			if !reflect.DeepEqual(test.err, err) {
				t.Errorf("error was %q instead of %q", err, test.err)
			}
			if test.clusterReturnIsNil && c != nil {
				t.Error("cluster was not nil")
			}
			if !test.clusterReturnIsNil && test.err == nil {
				if c.Status.ServiceAccountKey == nil {
					t.Error("service account key is nil")
				}
			}

		})
	}
}
