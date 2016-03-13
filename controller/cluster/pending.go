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
	"path"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster/template"
	"github.com/kubermatic/api/provider/kubernetes"
	"golang.org/x/crypto/ssh"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func (cc *clusterController) syncPendingCluster(c *api.Cluster) (*api.Cluster, error) {
	changedC, err := cc.checkTimeout(c)
	if err != nil {
		return nil, err
	}

	changedC, err = cc.pendingCreateRootCA(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// create token-users first and also persist immediately because this
	// changes the cluster. The later secrets and other resources don't.
	changedC, err = cc.launchingCheckTokenUsers(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// check that all services are available
	changedC, err = cc.launchingCheckServices(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	changedC, err = cc.pendingCheckSecrets(c)
	if err != nil || changedC != nil {
		return changedC, err
	}

	// check that the ingress is available
	err = cc.launchingCheckIngress(c)
	if err != nil {
		return nil, err
	}

	// check that all replication controllers are available
	err = cc.launchingCheckReplicationController(c)
	if err != nil {
		return nil, err
	}

	c.Status.LastTransitionTime = time.Now()
	c.Status.Phase = api.LaunchingClusterStatusPhase
	return c, nil
}

func (cc *clusterController) pendingCreateRootCA(c *api.Cluster) (*api.Cluster, error) {
	if c.Status.RootCA.Key != nil {
		return nil, nil
	}

	rootCAReq := csr.CertificateRequest{
		CN: fmt.Sprintf("root-ca."+cc.hostPattern, c.Metadata.Name, cc.dc),
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
		return nil, fmt.Errorf("error creating root-ca: %v", err)
	}

	return c, nil
}

func (cc *clusterController) pendingCheckSecrets(c *api.Cluster) (*api.Cluster, error) {
	createApiserverAuth := func(t *template.Template) (*api.Cluster, *kapi.Secret, error) {
		saKey, err := createServiceAccountKey()
		if err != nil {
			return nil, nil, err
		}

		asKC, err := c.CreateKeyCert("10.10.0.1")
		if err != nil {
			return nil, nil, err
		}

		data := struct {
			ApiserverKey, ApiserverCert, RootCACert, ServiceAccountKey string
		}{
			ApiserverKey:      asKC.Key.Base64(),
			ApiserverCert:     asKC.Cert.Base64(),
			RootCACert:        c.Status.RootCA.Cert.Base64(),
			ServiceAccountKey: saKey.Base64(),
		}
		var secret kapi.Secret
		err = t.Execute(data, &secret)
		return nil, &secret, err
	}

	createEtcdAuth := func(t *template.Template) (*api.Cluster, *kapi.Secret, error) {
		u, err := url.Parse(c.Address.EtcdURL)
		if err != nil {
			return nil, nil, err
		}
		etcdKC, err := c.CreateKeyCert(strings.Split(u.Host, ":")[0])
		if err != nil {
			return nil, nil, err
		}

		data := struct {
			EtcdKey, EtcdCert, RootCACert string
		}{
			RootCACert: c.Status.RootCA.Cert.Base64(),
			EtcdKey:    etcdKC.Key.Base64(),
			EtcdCert:   etcdKC.Cert.Base64(),
		}
		var secret kapi.Secret
		err = t.Execute(data, &secret)
		return nil, &secret, err
	}

	createApiserverSSH := func(t *template.Template) (*api.Cluster, *kapi.Secret, error) {
		kc, err := createSSHKeyCert()
		if err != nil {
			return nil, nil, err
		}

		data := struct {
			Key, Cert string
		}{
			Key:  kc.Key.Base64(),
			Cert: kc.Cert.Base64(),
		}
		var secret kapi.Secret
		err = t.Execute(data, &secret)
		if err != nil {
			return nil, nil, err
		}

		glog.Warningf("####################### %v ###############", len(kc.Cert))
		c.Status.ApiserverSSH = string(kc.Cert)

		return c, &secret, nil
	}

	secrets := map[string]func(t *template.Template) (*api.Cluster, *kapi.Secret, error){
		"apiserver-auth":   createApiserverAuth,
		"apiserver-ssh":    createApiserverSSH,
		"etcd-public-auth": createEtcdAuth,
	}

	recreateSecrets := map[string]struct{}{}
	if len(c.Status.ApiserverSSH) == 0 {
		recreateSecrets["apiserver-ssh"] = struct{}{}
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	for s, gen := range secrets {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.secretStore.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if exists {
			if _, recreate := recreateSecrets[s]; !recreate {
				glog.V(6).Infof("Skipping already existing secret %q", key)
				continue
			}

			err = cc.client.Secrets(ns).Delete(s)
			if err != nil {
				return nil, err
			}
		}

		t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-secret.yaml"))
		if err != nil {
			return nil, err
		}

		changedC, secret, err := gen(t)
		if err != nil {
			return nil, err
		}

		_, err = cc.client.Secrets(ns).Create(secret)
		if err != nil {
			return nil, err
		}

		cc.recordClusterEvent(c, "pending", "Created secret %q", key)

		if changedC != nil {
			return changedC, nil
		}
	}

	return nil, nil
}

func (cc *clusterController) launchingCheckTokenUsers(c *api.Cluster) (*api.Cluster, error) {
	generateTokenUsers := func() (*kapi.Secret, error) {
		rawToken := make([]byte, 64)
		_, err := crand.Read(rawToken)
		if err != nil {
			return nil, err
		}
		token := sha256.Sum256(rawToken)
		token64 := base64.URLEncoding.EncodeToString(token[:])
		trimmedToken64 := strings.TrimRight(token64, "=")

		secret := kapi.Secret{
			ObjectMeta: kapi.ObjectMeta{
				Name: "token-users",
			},
			Type: kapi.SecretTypeOpaque,
			Data: map[string][]byte{
				"file": []byte(fmt.Sprintf("%s,admin,admin", trimmedToken64)),
			},
		}

		c.Address = &api.ClusterAddress{
			URL:   fmt.Sprintf("https://"+cc.hostPattern, c.Metadata.Name, cc.dc),
			Token: trimmedToken64,
		}

		return &secret, nil
	}
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	key := fmt.Sprintf("%s/token-users", ns)
	_, exists, err := cc.secretStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if exists {
		glog.V(6).Infof("Skipping already existing secret %q", key)
		return nil, nil
	}

	secret, err := generateTokenUsers()
	if err != nil {
		return nil, err
	}
	_, err = cc.client.Secrets(ns).Create(secret)
	if err != nil {
		return nil, err
	}
	cc.recordClusterEvent(c, "launching", "Created secret %q", key)
	return c, nil
}

func (cc *clusterController) launchingCheckServices(c *api.Cluster) (*api.Cluster, error) {
	loadFile := func(s string) (*kapi.Service, error) {
		t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-service.yaml"))
		if err != nil {
			return nil, err
		}

		var service kapi.Service
		err = t.Execute(nil, &service)
		return &service, err
	}

	services := map[string]func(s string) (*kapi.Service, error){
		"etcd":             loadFile,
		"etcd-public":      loadFile,
		"apiserver":        loadFile,
		"apiserver-public": loadFile,
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	for s, gen := range services {
		key := fmt.Sprintf("%s/%s", ns, s)
		_, exists, err := cc.serviceStore.GetByKey(key)
		if err != nil {
			return nil, err
		}

		if exists {
			glog.V(6).Infof("Skipping already existing service %q", key)
			continue
		}

		services, err := gen(s)
		if err != nil {
			return nil, err
		}

		_, err = cc.client.Services(ns).Create(services)
		if err != nil {
			return nil, err
		}

		cc.recordClusterEvent(c, "launching", "Created service %q", s)
	}

	if c.Address.EtcdURL != "" {
		return nil, nil
	}

	p, err := servicePort(cc.serviceStore, ns+"/etcd-public", "clients")
	if err != nil {
		return nil, err
	}

	c.Address.EtcdURL = fmt.Sprintf(
		"https://"+cc.hostPattern+":%d",
		c.Metadata.Name, cc.dc, p,
	)

	return c, nil
}

func (cc *clusterController) launchingCheckIngress(c *api.Cluster) error {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)
	key := fmt.Sprintf("%s/%s", ns, "apiserver-public")
	_, exists, err := cc.ingressStore.GetByKey(key)
	if err != nil {
		return err
	}
	if exists {
		glog.V(6).Infof("Skipping already existing ingress %q", key)
		return nil
	}

	t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, "ingress.yaml"))
	if err != nil {
		return err
	}

	var ingress extensions.Ingress
	data := struct {
		DC          string
		ClusterName string
	}{
		DC:          cc.dc,
		ClusterName: c.Metadata.Name,
	}
	err = t.Execute(data, &ingress)
	if err != nil {
		return err
	}

	_, err = cc.client.Ingress(ns).Create(&ingress)
	if err != nil {
		return err
	}

	cc.recordClusterEvent(c, "launching", "Created ingress")
	return nil
}

func (cc *clusterController) launchingCheckReplicationController(c *api.Cluster) error {
	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	loadFile := func(s string) (*kapi.ReplicationController, error) {
		t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-rc.yaml"))
		if err != nil {
			return nil, err
		}

		var rc kapi.ReplicationController
		err = t.Execute(nil, &rc)
		return &rc, err
	}

	loadApiserver := func(s string) (*kapi.ReplicationController, error) {
		u, err := url.Parse(c.Address.URL)
		if err != nil {
			return nil, err
		}
		addrs, err := net.LookupHost(u.Host)
		if err != nil {
			return nil, err
		}

		data := struct {
			AdvertiseAddress string
		}{
			AdvertiseAddress: addrs[0],
		}

		t, err := template.ParseFiles(path.Join(cc.masterResourcesPath, s+"-rc.yaml"))
		if err != nil {
			return nil, err
		}

		var rc kapi.ReplicationController
		err = t.Execute(data, &rc)
		return &rc, err
	}

	rcs := map[string]func(s string) (*kapi.ReplicationController, error){
		"etcd":               loadFile,
		"etcd-public":        loadFile,
		"apiserver":          loadApiserver,
		"controller-manager": loadFile,
		"scheduler":          loadFile,
	}

	existingRCs, err := cc.rcStore.ByIndex("namespace", ns)
	if err != nil {
		return err
	}

	for s, gen := range rcs {
		exists := false
		for _, obj := range existingRCs {
			rc := obj.(*kapi.ReplicationController)
			if role, found := rc.Spec.Selector["role"]; found && role == s {
				exists = true
				break
			}
		}
		if exists {
			glog.V(7).Infof("Skipping already existing rc %q for cluster %q", s, c.Metadata.Name)
			continue
		}

		rc, err := gen(s)
		if err != nil {
			return err
		}

		_, err = cc.client.ReplicationControllers(ns).Create(rc)
		if err != nil {
			return err
		}

		cc.recordClusterEvent(c, "launching", "Created rc %q", s)
	}

	return nil
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
	return &api.KeyCert{privBuf.Bytes(), ssh.MarshalAuthorizedKey(pub)}, nil
}
