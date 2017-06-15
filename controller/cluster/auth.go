package cluster

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider/kubernetes"
	"golang.org/x/crypto/ssh"
)

const svcSAN string = "kubernetes.default"

func (cc *clusterController) pendingCreateRootCA(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.RootCA.Key != nil {
		return nil, nil
	}

	rootCAReq := csr.CertificateRequest{
		CN: fmt.Sprintf("root-ca.%s.%s.%s", c.Metadata.Name, cc.dc, cc.externalURL),
		KeyRequest: &csr.BasicKeyRequest{
			A: "rsa",
			S: 2048,
		},
		CA: &csr.CAConfig{
			Expiry: fmt.Sprintf("%dh", 24*365*10),
		},
	}
	var err error
	c.Status.RootCA.Cert, _, c.Status.RootCA.Key, err = initca.New(&rootCAReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	glog.V(4).Infof("Created root ca for %s", kubernetes.NamespaceName(c.Metadata.Name))
	return c, nil
}

func (cc *clusterController) pendingCreateTokens(c *api.Cluster) (*api.Cluster, error) {
	if c.Address.AdminToken != "" {
		return nil, nil
	}

	adminToken, err := generateRandomToken()
	if err != nil {
		return nil, err
	}
	kubeletToken, err := generateRandomToken()
	if err != nil {
		return nil, err
	}

	c.Address.AdminToken = adminToken
	c.Address.KubeletToken = kubeletToken

	glog.V(4).Infof("Created admin & kubelet token for %s", kubernetes.NamespaceName(c.Metadata.Name))
	return c, nil
}

func (cc *clusterController) pendingCreateCertificates(c *api.Cluster) (*api.Cluster, error) {
	var updated bool

	if c.Status.ApiserverCert.Key == nil {
		asKC, err := c.CreateKeyCert(c.Address.ExternalName, []string{c.Address.ExternalName, "10.10.10.1", svcSAN})
		if err != nil {
			return nil, fmt.Errorf("failed to create apiserver-key/cert: %v", err)
		}
		c.Status.ApiserverCert.Key = asKC.Key
		c.Status.ApiserverCert.Cert = asKC.Cert
		glog.V(4).Infof("Created apiserver certificate for %s", kubernetes.NamespaceName(c.Metadata.Name))
		updated = true
	}

	if c.Status.KubeletCert.Key == nil {
		kubeletKC, err := c.CreateKeyCert(c.Address.ExternalName, []string{c.Address.ExternalName, "10.10.10.1"})
		if err != nil {
			return nil, fmt.Errorf("failed to create kubelet-key/cert: %v", err)
		}
		c.Status.KubeletCert.Key = kubeletKC.Key
		c.Status.KubeletCert.Cert = kubeletKC.Cert
		glog.V(4).Infof("Created kubelet certificate for %s", kubernetes.NamespaceName(c.Metadata.Name))
		updated = true
	}

	if updated {
		return c, nil
	}

	return nil, nil
}

func (cc *clusterController) pendingCreateServiceAccountKey(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.ServiceAccountKey != nil {
		return nil, nil
	}

	key, err := createServiceAccountKey()
	if err != nil {
		return nil, fmt.Errorf("error creating service account key: %v", err)
	}
	c.Status.ServiceAccountKey = key
	glog.V(4).Infof("Created service account key for %s", kubernetes.NamespaceName(c.Metadata.Name))
	return c, nil
}

func (cc *clusterController) pendingCreateApiserverSSHKeys(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.ApiserverSSHKey.PublicKey != nil {
		return nil, nil
	}

	k, err := createSSHKey()
	if err != nil {
		return nil, fmt.Errorf("error creating service account key: %v", err)
	}

	c.Status.ApiserverSSHKey.PublicKey = k.PublicKey
	c.Status.ApiserverSSHKey.PrivateKey = k.PrivateKey

	//TODO: Deprecated: Remove at some point with Dashboard V2
	c.Status.ApiserverSSH = string(k.PublicKey)

	glog.V(4).Infof("Created apiserver ssh keys for %s", kubernetes.NamespaceName(c.Metadata.Name))
	return c, nil
}

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

func createSSHKey() (*api.SecretRSAKeys, error) {
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
	return &api.SecretRSAKeys{PrivateKey: privBuf.Bytes(), PublicKey: ssh.MarshalAuthorizedKey(pub)}, nil
}

func generateRandomToken() (string, error) {
	rawToken := make([]byte, 64)
	_, err := rand.Read(rawToken)
	return base64.StdEncoding.EncodeToString(rawToken), err
}
