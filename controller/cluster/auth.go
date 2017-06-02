package cluster

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/template"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

const svcSAN string = "kubernetes.default"

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

	asKC, err := c.CreateKeyCert(host, []string{host, "10.10.10.1", svcSAN})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create apiserver-key/cert: %v", err)
	}

	kubeletKC, err := c.CreateKeyCert(host, []string{host, "10.10.10.1"})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubelet-key/cert: %v", err)
	}

	data := struct {
		ApiserverKey, ApiserverCert, RootCACert, RootCAKey, ServiceAccountKey, KubeletClientKey, KubeletClientCert string
	}{
		ApiserverKey:      asKC.Key.Base64(),
		ApiserverCert:     asKC.Cert.Base64(),
		KubeletClientCert: kubeletKC.Cert.Base64(),
		KubeletClientKey:  kubeletKC.Key.Base64(),
		RootCACert:        c.Status.RootCA.Cert.Base64(),
		RootCAKey:         c.Status.RootCA.Key.Base64(),
		ServiceAccountKey: saKey.Base64(),
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

func generateRandomToken() (string, error) {
	rawToken := make([]byte, 64)
	_, err := rand.Read(rawToken)
	return base64.StdEncoding.EncodeToString(rawToken), err
}

func createTokens(c *api.Cluster) (*v1.Secret, error) {
	adminToken, err := generateRandomToken()
	if err != nil {
		return nil, err
	}
	kubeletToken, err := generateRandomToken()
	if err != nil {
		return nil, err
	}

	buffer := bytes.Buffer{}
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{kubeletToken, "kubelet-bootstrap", "10001", "system:kubelet-bootstrap"}); err != nil {
		return nil, err
	}
	if err := writer.Write([]string{adminToken, "admin", "10000", "admin"}); err != nil {
		return nil, err
	}
	writer.Flush()

	c.Address.KubeletToken = kubeletToken
	c.Address.AdminToken = adminToken

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "token-users",
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"tokens.csv": buffer.Bytes(),
		},
	}

	return &secret, nil
}
