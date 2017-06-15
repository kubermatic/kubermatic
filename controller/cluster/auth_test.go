package cluster

import (
	"github.com/kubermatic/api"
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
				PublicKey:  []byte("FOO"),
				PrivateKey: []byte("BAR"),
			},
		},
	}

	c, err := cc.pendingCreateRootCA(c)
	if err != nil {
		t.Fatal(err)
	}

	oldPrivKey := c.Status.ApiserverSSHKey.PrivateKey
	oldPubKey := c.Status.ApiserverSSHKey.PublicKey
	_, err = cc.pendingCreateApiserverSSHKeys(c)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(c.Status.ApiserverSSHKey.PublicKey) != string(oldPubKey) || string(c.Status.ApiserverSSHKey.PrivateKey) != string(oldPrivKey) {
		t.Fatalf("apiserver ssh key was overwritten")
	}

}
