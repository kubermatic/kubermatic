package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"
)

// AdminKubeconfig returns a secret with the AdminKubeconfig key
func AdminKubeconfig(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}

	se.Name = AdminKubeconfigSecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	ca, err := data.GetClusterCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}

	config := clientcmdapi.Config{
		CurrentContext: data.Cluster.Name,
		Clusters: map[string]*clientcmdapi.Cluster{
			data.Cluster.Name: {
				Server: data.Cluster.Address.URL,
				CertificateAuthorityData: certutil.EncodeCertPEM(ca.Cert),
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			data.Cluster.Name: {
				Cluster:  data.Cluster.Name,
				AuthInfo: data.Cluster.Name,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			data.Cluster.Name: {
				Token: data.Cluster.Address.AdminToken,
			},
		},
	}

	b, err := clientcmd.Write(config)
	if err != nil {
		return nil, err
	}

	se.Data = map[string][]byte{
		AdminKubeconfigSecretKey: b,
	}

	return se, nil
}

// GetInternalKubeconfigCreator is a generic function to return a secret generator to create a client certificate signed by the cluster CA
func GetInternalKubeconfigCreator(name, commonName string) func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}
		const dataKey = "kubeconfig"

		se.Name = name
		se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		ca, err := data.GetClusterCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster ca: %v", err)
		}

		url, err := data.GetInternalApiserverURL()
		if err != nil {
			return nil, fmt.Errorf("failed to get internal apiserver url: %v", err)
		}

		kubeconfig := &clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				"default": {
					CertificateAuthorityData: certutil.EncodeCertPEM(ca.Cert),
					Server: url,
				},
			},
			CurrentContext: "default",
			Contexts: map[string]*clientcmdapi.Context{
				"default": {
					Cluster:  "default",
					AuthInfo: "default",
				},
			},
		}

		kubeconfigWithNewCert := func() (*corev1.Secret, error) {
			kp, err := triple.NewClientKeyPair(ca, commonName, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s key pair: %v", name, err)
			}

			kubeconfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				"default": {
					ClientCertificateData: certutil.EncodeCertPEM(kp.Cert),
					ClientKeyData:         certutil.EncodePrivateKeyPEM(kp.Key),
				},
			}

			kb, err := clientcmd.Write(*kubeconfig)
			if err != nil {
				return nil, err
			}

			se.Data = map[string][]byte{
				"kubeconfig": kb,
			}
			return se, nil
		}

		if b, exists := se.Data[dataKey]; exists {
			existingKubeconfig, err := clientcmd.Load(b)
			if err != nil {
				return kubeconfigWithNewCert()
			}

			kubeconfig.AuthInfos = existingKubeconfig.AuthInfos
			if !equality.Semantic.DeepEqual(kubeconfig, existingKubeconfig) {
				return kubeconfigWithNewCert()
			}

			certs, err := certutil.ParseCertsPEM(existingKubeconfig.AuthInfos["default"].ClientCertificateData)
			if err != nil {
				return kubeconfigWithNewCert()
			}

			if !IsClientCertificateValidForAllOf(certs[0], commonName, nil) {
				return kubeconfigWithNewCert()
			}

			return se, nil
		}

		return kubeconfigWithNewCert()
	}
}
