package cluster

import (
	"strings"
	"testing"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/template"
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
	k, err := createSSHKeyCert()

	if err != nil {
		t.Fatal("Unexpected error: ", err)
	}

	if len(k.Key) == 0 {
		t.Fatalf("Expected byte buffer in Key to have length > 0, got %d", len(k.Key))
	}

	if len(k.Cert) == 0 {
		t.Fatalf("Expected byte buffer in Cert to have length > 0, got %d", len(k.Cert))
	}

	ks := string(k.Key)
	header := "RSA PRIVATE KEY"
	if !strings.Contains(ks, header) {
		t.Fatalf("Expected retured k.Key byte buffer to contain %s", header)
	}

	cs := string(k.Cert)
	cHeader := "ssh-rsa"
	if !strings.Contains(cs, cHeader) {
		t.Fatalf("Expected retured k.Key byte buffer to contain %s", cHeader)
	}
}

func TestCreateApiserverAuth(t *testing.T) {
	_, cc := newTestController()
	cl := &api.Cluster{
		Address: &api.ClusterAddress{
			URL: "https://asdf.test-de-01.kubermatic.io:8443",
		},
		Metadata: api.Metadata{
			Name: "asdf",
		},
	}

	cl, err := cc.pendingCreateRootCA(cl)
	if err != nil {
		t.Fatal(err)
	}

	temp := readTestFile("./fixtures/templates/apiserver-auth-secret.yaml")
	_, secret, err := createApiserverAuth(cc, cl, temp)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, has := secret.Data["apiserver.crt"]; !has {
		t.Error("secret data does not contain apiserver.crt")
	}
	if _, has := secret.Data["apiserver.key"]; !has {
		t.Error("secret data does not contain apiserver.key")
	}
	if _, has := secret.Data["root-ca.crt"]; !has {
		t.Error("secret data does not contain root-ca.crt")
	}
	if _, has := secret.Data["service-account.key"]; !has {
		t.Error("secret data does not contain service-account.key")
	}
}

func readTestFile(path string) *template.Template {

	temp, err := template.ParseFiles(path)
	if err != nil {
		panic(err)
	}

	return temp
}
