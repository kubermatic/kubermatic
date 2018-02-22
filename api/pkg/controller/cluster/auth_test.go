package cluster

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
			ApiserverSSHKey: kubermaticv1.RSAKeys{},
		},
	}

	if err := f.controller.ensureRootCA(c); err != nil {
		t.Fatal(err)
	}

	if err := f.controller.ensureApiserverSSHKeypair(c); err != nil {
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
			ApiserverSSHKey: kubermaticv1.RSAKeys{
				PublicKey:  []byte("PUB_KEY"),
				PrivateKey: []byte("PRIV_KEY"),
			},
		},
	}

	if err := f.controller.ensureRootCA(c); err != nil {
		t.Fatal(err)
	}

	if err := f.controller.ensureApiserverSSHKeypair(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(c.Status.ApiserverSSHKey.PublicKey) != "PUB_KEY" || string(c.Status.ApiserverSSHKey.PrivateKey) != "PRIV_KEY" {
		t.Fatalf("apiserver ssh key was overwritten")
	}

}

func TestPendingCreateCertificates(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	tests := []struct {
		name         string
		cluster      *kubermaticv1.Cluster
		err          error
		createRootCA bool
	}{
		{
			name:         "successful both",
			err:          nil,
			createRootCA: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:         "successful partial apiserver",
			err:          nil,
			createRootCA: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
				},
			},
		},
		{
			name:         "successful partial kubelet",
			err:          nil,
			createRootCA: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{Key: []byte("foo"), Cert: []byte("bar")},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:         "root-ca is missing",
			err:          errors.New("failed to parse root-ca cert: data does not contain any valid RSA or ECDSA certificates"),
			createRootCA: false,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:         "external name is missing",
			err:          errors.New("external name is undefined"),
			createRootCA: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
					ApiserverCert: kubermaticv1.KeyCert{},
					KubeletCert:   kubermaticv1.KeyCert{},
				},
			},
		},
		{
			name:         "already exists",
			err:          nil,
			createRootCA: true,
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "henrik1",
				},
				Address: &kubermaticv1.ClusterAddress{
					ExternalName: "6vcgjl87w.us-central1.develop.kubermatic.io",
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA:        kubermaticv1.KeyCert{},
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
				if err = f.controller.ensureRootCA(test.cluster); err != nil {
					t.Errorf("could not create root ca")
				}
			}
			err = f.controller.ensureCertificates(test.cluster)
			if !reflect.DeepEqual(test.err, err) {
				t.Errorf("error was %q instead of %q", err, test.err)
			}
			if test.err == nil {
				if test.cluster.Status.ApiserverCert.Cert == nil {
					t.Error("apiserver cert is nil")
				}
				if test.cluster.Status.ApiserverCert.Key == nil {
					t.Error("apiserver cert key is nil")
				}
				if test.cluster.Status.KubeletCert.Cert == nil {
					t.Error("kubelet cert is nil")
				}
				if test.cluster.Status.KubeletCert.Key == nil {
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
			RootCA: kubermaticv1.KeyCert{},
		},
	}

	if err := f.controller.ensureRootCA(c); err != nil {
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
			RootCA: kubermaticv1.KeyCert{
				Cert: kubermaticv1.Bytes([]byte("CERT")),
				Key:  kubermaticv1.Bytes([]byte("KEY")),
			},
		},
	}

	if err := f.controller.ensureRootCA(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
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

	if err := f.controller.ensureTokens(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Address.KubeletToken == "" {
		t.Fatalf("kubelet token is empty")
	}

	if c.Address.AdminToken == "" {
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

	if err := f.controller.ensureTokens(c); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Address.KubeletToken != "KubeletToken" || c.Address.AdminToken != "AdminToken" {
		t.Fatalf("tokens were overwritten")
	}
}

func TestPendingCreateServiceAccountKey(t *testing.T) {
	f := newTestController([]runtime.Object{}, []runtime.Object{}, []runtime.Object{})
	tests := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		err     error
	}{
		{
			name: "successful",
			err:  nil,
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
			name: "already exists",
			err:  nil,
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
			err := f.controller.ensureCreateServiceAccountKey(test.cluster)
			if !reflect.DeepEqual(test.err, err) {
				t.Errorf("error was %q instead of %q", err, test.err)
			}
			if test.err == nil && test.cluster.Status.ServiceAccountKey == nil {
				t.Error("service account key is nil")
			}
		})
	}
}
