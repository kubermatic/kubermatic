package cluster

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"golang.org/x/crypto/ssh"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

func (cc *controller) pendingCreateRootCA(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if c.Status.RootCA.Key != nil {
		return nil, nil
	}

	k, err := triple.NewCA(fmt.Sprintf("root-ca.%s.%s.%s", c.Name, c.Spec.SeedDatacenterName, cc.externalURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	c.Status.RootCA.Key = cert.EncodePrivateKeyPEM(k.Key)
	c.Status.RootCA.Cert = cert.EncodeCertPEM(k.Cert)

	glog.V(4).Infof("Created root ca for %s", kubernetes.NamespaceName(c.Name))
	return c, nil
}

func (cc *controller) pendingCreateTokens(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var updated bool

	if c.Address.AdminToken == "" {
		adminToken, err := generateRandomToken()
		if err != nil {
			return nil, err
		}
		c.Address.AdminToken = adminToken
		glog.V(4).Infof("Created admin token for %s", kubernetes.NamespaceName(c.Name))
		updated = true
	}

	if c.Address.KubeletToken == "" {
		kubeletToken, err := generateRandomToken()
		if err != nil {
			return nil, err
		}
		c.Address.KubeletToken = kubeletToken
		glog.V(4).Infof("Created kubelet token for %s", kubernetes.NamespaceName(c.Name))
		updated = true
	}

	if updated {
		return c, nil
	}
	return nil, nil
}

func (cc *controller) pendingCreateCertificates(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var updated bool

	if c.Address.ExternalName == "" {
		return nil, errors.New("external name is undefined")
	}

	certs, err := cert.ParseCertsPEM(c.Status.RootCA.Cert)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root-ca cert: %v", err)
	}

	key, err := cert.ParsePrivateKeyPEM(c.Status.RootCA.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root-ca key: %v", err)
	}

	caKp := &triple.KeyPair{
		Cert: certs[0],
		Key:  key.(*rsa.PrivateKey),
	}

	if c.Status.ApiserverCert.Key == nil {
		apiKp, err := triple.NewServerKeyPair(caKp, c.Address.ExternalName, "kubernetes", "default", "cluster.local", []string{"10.10.10.1"}, []string{c.Address.ExternalName})
		if err != nil {
			return nil, fmt.Errorf("failed to create apiserver key pair: %v", err)
		}

		c.Status.ApiserverCert.Key = cert.EncodePrivateKeyPEM(apiKp.Key)
		c.Status.ApiserverCert.Cert = cert.EncodeCertPEM(apiKp.Cert)
		glog.V(4).Infof("Created apiserver certificate for %s", kubernetes.NamespaceName(c.Name))
		updated = true
	}

	if c.Status.KubeletCert.Key == nil {
		kubeletKp, err := triple.NewClientKeyPair(caKp, c.Address.ExternalName, []string{c.Address.ExternalName})
		if err != nil {
			return nil, fmt.Errorf("failed to create kubelet client key pair: %v", err)
		}

		c.Status.KubeletCert.Key = cert.EncodePrivateKeyPEM(kubeletKp.Key)
		c.Status.KubeletCert.Cert = cert.EncodeCertPEM(kubeletKp.Cert)
		glog.V(4).Infof("Created kubelet certificate for %s", kubernetes.NamespaceName(c.Name))
		updated = true
	}

	if updated {
		return c, nil
	}

	return nil, nil
}

func (cc *controller) pendingCreateServiceAccountKey(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if c.Status.ServiceAccountKey != nil {
		return nil, nil
	}

	key, err := createServiceAccountKey()
	if err != nil {
		return nil, fmt.Errorf("error creating service account key: %v", err)
	}
	c.Status.ServiceAccountKey = key
	glog.V(4).Infof("Created service account key for %s", kubernetes.NamespaceName(c.Name))
	return c, nil
}

func (cc *controller) pendingCreateApiserverSSHKeys(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if c.Status.ApiserverSSHKey.PublicKey != nil {
		return nil, nil
	}

	k, err := createSSHKey()
	if err != nil {
		return nil, fmt.Errorf("error creating service account key: %v", err)
	}

	c.Status.ApiserverSSHKey.PublicKey = k.PublicKey
	c.Status.ApiserverSSHKey.PrivateKey = k.PrivateKey

	glog.V(4).Infof("Created apiserver ssh keys for %s", kubernetes.NamespaceName(c.Name))
	return c, nil
}

func createServiceAccountKey() (kubermaticv1.Bytes, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	saKey := x509.MarshalPKCS1PrivateKey(priv)
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: saKey,
	}
	return kubermaticv1.Bytes(pem.EncodeToMemory(&block)), nil
}

func createSSHKey() (*kubermaticv1.RSAKeys, error) {
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
	return &kubermaticv1.RSAKeys{PrivateKey: privBuf.Bytes(), PublicKey: ssh.MarshalAuthorizedKey(pub)}, nil
}

func generateRandomToken() (string, error) {
	rawToken := make([]byte, 64)
	_, err := rand.Read(rawToken)
	return base64.StdEncoding.EncodeToString(rawToken), err
}
