package cluster

import (
	"bytes"
	"crypto/rand"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster/template"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/pkg/api/v1"
)

func createServiceAccountKey() (api.Bytes, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	saKey := x509.MarshalPKCS1PrivateKey(priv)
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: saKey,
	}
	return api.Bytes(pem.EncodeToMemory(&block)), nil
}

func createSSHKeyCert() (*api.KeyCert, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	privBuf := bytes.Buffer{}
	err = pem.Encode(&privBuf, privateKeyPEM)
	if err != nil {
		return nil, err
	}

	pub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	return &api.KeyCert{Key: privBuf.Bytes(), Cert: ssh.MarshalAuthorizedKey(pub)}, nil
}

func createApiserverAuth(cc *clusterController, c *api.Cluster, t *template.Template) (*api.Cluster, *v1.Secret, error) {
	saKey, err := createServiceAccountKey()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating service account key: %v", err)
	}

	u, err := url.Parse(c.Address.URL)
	if err != nil {
		return nil, nil, err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, nil, err
	}

	asKC, err := c.CreateKeyCert(host, []string{host, "10.10.10.1"})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create apiserver-key/cert: %v", err)
	}

	kKC, err := c.CreateKeyCert(host, []string{host, "10.10.10.1"})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubelet-key/cert: %v", err)
	}

	data := struct {
		ApiserverKey, ApiserverCert, RootCACert, ServiceAccountKey, KubeletKey, KubeletCert string
	}{
		ApiserverKey:      asKC.Key.Base64(),
		ApiserverCert:     asKC.Cert.Base64(),
		KubeletCert:       kKC.Cert.Base64(),
		KubeletKey:        kKC.Key.Base64(),
		RootCACert:        c.Status.RootCA.Cert.Base64(),
		ServiceAccountKey: saKey.Base64(),
	}
	var secret v1.Secret
	err = t.Execute(data, &secret)
	return nil, &secret, err
}

func createEtcdAuth(cc *clusterController, c *api.Cluster, t *template.Template) (*api.Cluster, *v1.Secret, error) {
	u, err := url.Parse(c.Address.EtcdURL)
	if err != nil {
		return nil, nil, err
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, nil, err
	}

	etcdKC, err := c.CreateKeyCert(host, []string{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create key cert: %v", err)
	}

	data := struct {
		EtcdKey, EtcdCert, RootCACert string
	}{
		RootCACert: c.Status.RootCA.Cert.Base64(),
		EtcdKey:    etcdKC.Key.Base64(),
		EtcdCert:   etcdKC.Cert.Base64(),
	}
	var secret v1.Secret
	err = t.Execute(data, &secret)
	return nil, &secret, err
}

func createApiserverSSH(cc *clusterController, c *api.Cluster, t *template.Template) (*api.Cluster, *v1.Secret, error) {
	kc, err := createSSHKeyCert()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating service account key: %v", err)
	}

	data := struct {
		Key, Cert string
	}{
		Key:  kc.Key.Base64(),
		Cert: kc.Cert.Base64(),
	}
	var secret v1.Secret
	err = t.Execute(data, &secret)
	if err != nil {
		return nil, nil, err
	}

	glog.Warningf("####################### %v ###############", len(kc.Cert))
	c.Status.ApiserverSSH = string(kc.Cert)

	return c, &secret, nil
}

func generateTokenUsers(cc *clusterController, c *api.Cluster) (*v1.Secret, error) {
	rawToken := make([]byte, 64)
	_, err := crand.Read(rawToken)
	if err != nil {
		return nil, err
	}
	token := sha256.Sum256(rawToken)
	token64 := base64.URLEncoding.EncodeToString(token[:])
	trimmedToken64 := strings.TrimRight(token64, "=")

	secret := v1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name: "token-users",
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"file": []byte(fmt.Sprintf("%s,admin,admin", trimmedToken64)),
		},
	}

	c.Address.URL = fmt.Sprintf("https://%s.%s.%s:8443", c.Metadata.Name, cc.dc, cc.externalURL)
	c.Address.Token = trimmedToken64

	return &secret, nil
}
