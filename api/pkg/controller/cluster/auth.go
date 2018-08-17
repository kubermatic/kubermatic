package cluster

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

func (cc *Controller) secretWithData(data map[string][]byte, c *kubermaticv1.Cluster) *corev1.Secret {

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:     map[string]string{},
			Labels:          map[string]string{},
			OwnerReferences: []metav1.OwnerReference{cc.getOwnerRefForCluster(c)},
		},
		Data: data,
	}
}

func (cc *Controller) secretWithJSON(secret *corev1.Secret) (*corev1.Secret, string, error) {
	b, err := json.Marshal(secret)
	if err != nil {
		return nil, "", fmt.Errorf("unable to marshal secret to JSON: %v", err)
	}
	return secret, string(b), nil
}

func (cc *Controller) createCASecret(commonName string, c *kubermaticv1.Cluster) (map[string][]byte, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a private key for new CA: %v", err)
	}

	config := certutil.Config{CommonName: commonName}

	caCert, err := certutil.NewSelfSignedCACert(config, key)
	if err != nil {
		return nil, fmt.Errorf("unable to create a self-signed certificate for a new CA: %v", err)
	}

	return map[string][]byte{
		resources.CACertSecretKey: certutil.EncodeCertPEM(caCert),
		resources.CAKeySecretKey:  certutil.EncodePrivateKeyPEM(key),
	}, nil
}

func (cc *Controller) getImagePullSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	kubermaticDockerCfg, err := cc.secretLister.Secrets(resources.KubermaticNamespaceName).Get(resources.ImagePullSecretName)
	if err != nil {
		return nil, "", fmt.Errorf("couldn't retrieve dockercfg from kubermatic ns: %v", err)
	}

	var secret *corev1.Secret
	if existingSecret != nil {
		secret = existingSecret
	} else {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Annotations:     map[string]string{},
				Labels:          map[string]string{},
				OwnerReferences: []metav1.OwnerReference{cc.getOwnerRefForCluster(c)},
			},
			Type: corev1.SecretTypeDockerConfigJson,
		}
	}

	secret.Data = kubermaticDockerCfg.Data

	return cc.secretWithJSON(secret)
}

func (cc *Controller) getRootCACertSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	if existingSecret == nil {
		data, err := cc.createCASecret(fmt.Sprintf("root-ca.%s.%s.%s", c.Name, cc.dc, cc.externalURL), c)
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a self-signed certificate for a new CA: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}
	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) getFrontProxyCACertSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	if existingSecret == nil {
		data, err := cc.createCASecret("front-proxy-ca", c)
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a self-signed certificate for a new CA: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}
	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) getFullCAFromLister(name string, c *kubermaticv1.Cluster) (*triple.KeyPair, error) {
	caSecret, err := cc.secretLister.Secrets(c.Status.NamespaceName).Get(name)
	if err != nil {
		return nil, fmt.Errorf("unable to check if a CA cert already exists: %v", err)
	}

	certs, err := certutil.ParseCertsPEM(caSecret.Data[resources.CACertSecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid cert from the CA secret %s: %v", resources.CASecretName, err)
	}

	key, err := certutil.ParsePrivateKeyPEM(caSecret.Data[resources.CAKeySecretKey])
	if err != nil {
		return nil, fmt.Errorf("got an invalid private key from the CA secret %s: %v", resources.CASecretName, err)
	}

	return &triple.KeyPair{
		Cert: certs[0],
		Key:  key.(*rsa.PrivateKey),
	}, nil
}

func (cc *Controller) createApiserverTLSCertificatesSecret(caKp *triple.KeyPair, commonName, svcName, svcNamespace, dnsDomain string, ips, hostnames []string) (map[string][]byte, error) {
	apiKp, err := triple.NewServerKeyPair(caKp, commonName, svcName, svcNamespace, dnsDomain, ips, hostnames)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiserver key pair: %v", err)
	}

	return map[string][]byte{
		resources.ApiserverTLSKeySecretKey:  certutil.EncodePrivateKeyPEM(apiKp.Key),
		resources.ApiserverTLSCertSecretKey: certutil.EncodeCertPEM(apiKp.Cert),
	}, nil
}

func (cc *Controller) getApiserverServingCertificatesSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get CA: %v", err)
	}

	apiAddress, err := cc.getSecureApiserverAddress(c)

	commonName := c.Address.ExternalName
	svcName := "kubernetes"
	svcNamespace := "default"
	dnsDomain := "cluster.local"
	ips := sets.NewString("10.10.10.1", c.Address.IP, strings.Split(apiAddress, ":")[0])
	hostnames := sets.NewString(c.Address.ExternalName)

	if existingSecret == nil {
		data, err := cc.createApiserverTLSCertificatesSecret(caKp, commonName, svcName, svcNamespace, dnsDomain, ips.List(), hostnames.List())
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a apiserver tls certificates: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}

	// Validate that the certificate is up to date. Its safe to regenerate it. The apiserver will get automatically restarted when the secret gets updated
	b := existingSecret.Data[resources.ApiserverTLSCertSecretKey]
	certs, err := certutil.ParseCertsPEM(b)
	if err != nil {
		return nil, "", err
	}
	cert := certs[0]

	getIPStrings := func(inIps []net.IP) []string {
		s := make([]string, len(inIps))
		for i, ip := range inIps {
			s[i] = ip.String()
		}
		return s
	}

	differentCommonName := cert.Subject.CommonName != commonName
	differentIPs := !sets.NewString(getIPStrings(cert.IPAddresses)...).Equal(ips)
	dnsDomains := sets.NewString(commonName, svcName, svcName+"."+svcNamespace, svcName+"."+svcNamespace+".svc", svcName+"."+svcNamespace+".svc."+dnsDomain)
	differentDNSNames := !sets.NewString(cert.DNSNames...).Equal(dnsDomains)

	if differentCommonName || differentIPs || differentDNSNames {
		data, err := cc.createApiserverTLSCertificatesSecret(caKp, commonName, svcName, svcNamespace, dnsDomain, ips.List(), hostnames.List())
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a apiserver tls certificates: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}

	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) getKubeletClientCertificatesSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get CA: %v", err)
	}

	commonName := c.Address.ExternalName
	organizations := sets.NewString(commonName, "system:masters")

	if existingSecret == nil {
		data, err := cc.createKubeletClientCertificates(caKp, commonName, organizations.List())
		if err != nil {
			return nil, "", fmt.Errorf("failed to create kubelet client key pair: %v", err)
		}

		return cc.secretWithJSON(cc.secretWithData(data, c))
	}

	// Validate that the certificate is up to date. Its safe to regenerate it. The apiserver will get automatically restarted when the secret gets updated
	b := existingSecret.Data[resources.KubeletClientCertSecretKey]
	cert, err := certutil.ParseCertsPEM(b)
	if err != nil {
		return nil, "", err
	}
	if !organizations.Equal(sets.NewString(cert[0].Subject.Organization...)) {
		data, err := cc.createKubeletClientCertificates(caKp, commonName, organizations.List())
		if err != nil {
			return nil, "", fmt.Errorf("failed to create kubelet client key pair: %v", err)
		}

		return cc.secretWithJSON(cc.secretWithData(data, c))
	}

	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) createKubeletClientCertificates(caKp *triple.KeyPair, commonName string, organizations []string) (map[string][]byte, error) {
	kubeletKp, err := triple.NewClientKeyPair(caKp, commonName, organizations)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubelet client key pair: %v", err)
	}

	return map[string][]byte{
		resources.KubeletClientKeySecretKey:  certutil.EncodePrivateKeyPEM(kubeletKp.Key),
		resources.KubeletClientCertSecretKey: certutil.EncodeCertPEM(kubeletKp.Cert),
	}, nil
}

func (cc *Controller) createAdminKubeconfigSecret(c *kubermaticv1.Cluster) (map[string][]byte, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, fmt.Errorf("unable to get CA: %v", err)
	}

	config := clientcmdapi.Config{
		CurrentContext: c.Name,
		Clusters: map[string]*clientcmdapi.Cluster{
			c.Name: {
				Server: c.Address.URL,
				CertificateAuthorityData: certutil.EncodeCertPEM(caKp.Cert),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			c.Name: {
				Cluster:  c.Name,
				AuthInfo: c.Name,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			c.Name: {
				Token: c.Address.AdminToken,
			},
		},
	}

	b, err := clientcmd.Write(config)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		resources.AdminKubeconfigSecretKey: b,
	}, nil
}

func (cc *Controller) getAdminKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	//Its save to always generate it.
	data, err := cc.createAdminKubeconfigSecret(c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create a admin kubeconfig: %v", err)
	}
	return cc.secretWithJSON(cc.secretWithData(data, c))
}

func (cc *Controller) createTokenUsersSecret(c *kubermaticv1.Cluster) (map[string][]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write([]string{c.Address.AdminToken, "admin", "10000", "system:masters"}); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return map[string][]byte{
		resources.TokensSecretKey: buffer.Bytes(),
	}, nil
}

func (cc *Controller) getTokenUsersSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	//Its save to always generate it.
	data, err := cc.createTokenUsersSecret(c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to create a token users secret: %v", err)
	}
	return cc.secretWithJSON(cc.secretWithData(data, c))
}

func (cc *Controller) createOpenVPNServerCertificates(c *kubermaticv1.Cluster) (map[string][]byte, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, fmt.Errorf("unable to get CA: %v", err)
	}

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a openvpn server private key: %v", err)
	}

	config := certutil.Config{
		CommonName: "openvpn-server",
		AltNames:   certutil.AltNames{},
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	cert, err := certutil.NewSignedCert(config, key, caKp.Cert, caKp.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the openvpn server certificate: %v", err)
	}

	return map[string][]byte{
		resources.OpenVPNServerKeySecretKey:  certutil.EncodePrivateKeyPEM(key),
		resources.OpenVPNServerCertSecretKey: certutil.EncodeCertPEM(cert),
	}, nil
}

func (cc *Controller) getOpenVPNServerCertificates(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	if existingSecret == nil {
		data, err := cc.createOpenVPNServerCertificates(c)
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a openvpn server certificate: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}
	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) createOpenVPNInternalClientCertificates(c *kubermaticv1.Cluster) (map[string][]byte, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, fmt.Errorf("unable to get CA: %v", err)
	}

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("unable to create a openvpn client private key: %v", err)
	}

	config := certutil.Config{
		CommonName: "internal-client",
		AltNames:   certutil.AltNames{},
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := certutil.NewSignedCert(config, key, caKp.Cert, caKp.Key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign the openvpn client certificate: %v", err)
	}

	return map[string][]byte{
		resources.OpenVPNInternalClientKeySecretKey:  certutil.EncodePrivateKeyPEM(key),
		resources.OpenVPNInternalClientCertSecretKey: certutil.EncodeCertPEM(cert),
	}, nil
}

func (cc *Controller) getOpenVPNInternalClientCertificates(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	if existingSecret == nil {
		data, err := cc.createOpenVPNInternalClientCertificates(c)
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a openvpn client certificate: %v", err)
		}
		return cc.secretWithJSON(cc.secretWithData(data, c))
	}
	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func (cc *Controller) getSchedulerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername)
}

func (cc *Controller) getControllerManagerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername)
}

func (cc *Controller) getMachineControllerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername)
}

func (cc *Controller) getIPAMControllerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.IPAMControllerKubeconfigSecretName, resources.IPAMControllerCertUsername)
}

func (cc *Controller) getKubeStateMetricsKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername)
}

func (cc *Controller) getKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret, secretName, username string) (*corev1.Secret, string, error) {
	caKp, err := cc.getFullCAFromLister(resources.CASecretName, c)
	if err != nil {
		return nil, "", fmt.Errorf("unable to get CA: %v", err)
	}

	masterAddress, err := cc.getSecureApiserverAddress(c)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve apiserver service to master address: %v", err)
	}

	if existingSecret == nil {
		kconf, err := createLimitedKubeconfig(fmt.Sprintf("https://%s", masterAddress), caKp, username, []string{})
		if err != nil {
			return nil, "", fmt.Errorf("unable to create a dedicated kubeconfig for %s: %v", username, err)
		}
		return cc.secretWithJSON(cc.secretWithData(map[string][]byte{secretName: kconf}, c))
	}

	// FIXME add better reconcile handling.

	return cc.secretWithJSON(cc.secretWithData(existingSecret.Data, c))
}

func createLimitedKubeconfig(address string, ca *triple.KeyPair, commonName string, organizations []string) ([]byte, error) {
	kp, err := triple.NewClientKeyPair(ca, commonName, organizations)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create client certificates for kubeconfig: %v", err)
	}
	kubeconfig := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"default": {
				CertificateAuthorityData: certutil.EncodeCertPEM(ca.Cert),
				Server: address,
			},
		},
		CurrentContext: "default",
		Contexts: map[string]*clientcmdapi.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"default": {
				ClientCertificateData: certutil.EncodeCertPEM(kp.Cert),
				ClientKeyData:         certutil.EncodePrivateKeyPEM(kp.Key),
			},
		},
	}
	kb, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return []byte{}, err
	}
	return kb, nil
}

func (cc *Controller) getSecureApiserverAddress(c *kubermaticv1.Cluster) (string, error) {
	// Create a fake TemplateData for now, as it conveniently holds
	// a Cluster and a ServiceLister for us.
	tdata := &resources.TemplateData{Cluster: c, ServiceLister: cc.serviceLister}
	return tdata.InClusterApiserverAddress()
}
