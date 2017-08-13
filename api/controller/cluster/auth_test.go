package cluster

import (
	"github.com/kubermatic/kubermatic/api"
	"strings"
	"testing"
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
	_, cc := newTestController()
	c := &api.Cluster{
		Status: api.ClusterStatus{
			ApiserverSSHKey: api.SecretRSAKeys{},
		},
	}

	c, err := cc.pendingCreateRootCA(c)
	if err != nil {
		t.Fatal(err)
	}

	c, err = cc.pendingCreateApiserverSSHKeys(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if c.Status.ApiserverSSH == "" {
		t.Fatalf("deprecated apiserver ssh public key was not generated")
	}
	if c.Status.ApiserverSSHKey.PrivateKey == nil {
		t.Fatalf("apiserver ssh private key was not generated")
	}
	if c.Status.ApiserverSSHKey.PublicKey == nil {
		t.Fatalf("apiserver ssh public key was not generated")
	}

}

func TestPendingCreateApiserverSSHKeysAlreadyExist(t *testing.T) {
	_, cc := newTestController()
	c := &api.Cluster{
		Status: api.ClusterStatus{
			ApiserverSSHKey: api.SecretRSAKeys{
				PublicKey:  []byte("PUB_KEY"),
				PrivateKey: []byte("PRIV_KEY"),
			},
		},
	}

	c, err := cc.pendingCreateRootCA(c)
	if err != nil {
		t.Fatal(err)
	}

	changedC, err := cc.pendingCreateApiserverSSHKeys(c)
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

func TestPendingCreateRootCASuccessfully(t *testing.T) {
	_, cc := newTestController()
	c := &api.Cluster{
		Status: api.ClusterStatus{
			RootCA: api.SecretKeyCert{},
		},
	}

	c, err := cc.pendingCreateRootCA(c)
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
	_, cc := newTestController()
	c := &api.Cluster{
		Status: api.ClusterStatus{
			RootCA: api.SecretKeyCert{
				Cert: api.Bytes([]byte("CERT")),
				Key:  api.Bytes([]byte("KEY")),
			},
		},
	}

	changedC, err := cc.pendingCreateRootCA(c)
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
	_, cc := newTestController()
	c := &api.Cluster{
		Address: &api.ClusterAddress{},
	}

	changedC, err := cc.pendingCreateTokens(c)
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
	_, cc := newTestController()
	c := &api.Cluster{
		Address: &api.ClusterAddress{
			KubeletToken: "KubeletToken",
			AdminToken:   "AdminToken",
		},
	}

	changedC, err := cc.pendingCreateTokens(c)
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
