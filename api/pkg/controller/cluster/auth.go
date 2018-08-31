package cluster

import (
	"encoding/json"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (cc *Controller) createAdminKubeconfigSecret(c *kubermaticv1.Cluster) (map[string][]byte, error) {
	caKp, err := resources.GetClusterRootCA(c, cc.secretLister)
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

func (cc *Controller) getSchedulerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername)
}

func (cc *Controller) getKubeletDnatControllerKubeconfigSecret(c *kubermaticv1.Cluster, existingSecret *corev1.Secret) (*corev1.Secret, string, error) {
	return cc.getKubeconfigSecret(c, existingSecret, resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername)
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
	caKp, err := resources.GetClusterRootCA(c, cc.secretLister)
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
