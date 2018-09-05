package resources

import (
	"crypto/x509"
	"fmt"

	"github.com/golang/glog"
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

	if se.Data == nil {
		se.Data = map[string][]byte{}
	}

	ca, err := data.GetRootCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster ca: %v", err)
	}

	config := getBaseKubeconfig(ca.Cert, data.Cluster.Address.URL, data.Cluster.Name)
	config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		KubeconfigDefaultContextKey: {
			Token: data.Cluster.Address.AdminToken,
		},
	}

	b, err := clientcmd.Write(*config)
	if err != nil {
		return nil, err
	}

	se.Data[KubeconfigSecretKey] = b

	return se, nil
}

// GetInternalKubeconfigCreator is a generic function to return a secret generator to create a kubeconfig which must only be used within the seed-cluster as it uses the ClusterIP of the apiserver.
func GetInternalKubeconfigCreator(name, commonName string, organizations []string) func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
		var se *corev1.Secret
		if existing != nil {
			se = existing
		} else {
			se = &corev1.Secret{}
		}

		se.Name = name
		se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		if se.Data == nil {
			se.Data = map[string][]byte{}
		}

		ca, err := data.GetRootCA()
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster ca: %v", err)
		}

		url, err := data.InClusterApiserverURL()
		if err != nil {
			return nil, fmt.Errorf("failed to get internal apiserver url: %v", err)
		}

		b := se.Data[KubeconfigSecretKey]
		valid, err := isValidKubeconfig(b, ca.Cert, url.String(), commonName, data.Cluster.Name)
		if err != nil || !valid {
			if err != nil {
				glog.V(2).Infof("failed to validate existing kubeconfig from %s/%s %v. Regenerating it...", se.Namespace, se.Name, err)
			} else {
				glog.V(2).Infof("invalid/outdated kubeconfig found in %s/%s. Regenerating it...", se.Namespace, se.Name)
			}

			se.Data[KubeconfigSecretKey], err = buildNewKubeconfigAsByte(ca, url.String(), commonName, organizations, data.Cluster.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create new kubeconfig: %v", err)
			}
			return se, nil
		}

		return se, nil
	}
}

func buildNewKubeconfigAsByte(ca *triple.KeyPair, server, commonName string, organizations []string, clusterName string) ([]byte, error) {
	kubeconfig, err := buildNewKubeconfig(ca, server, commonName, organizations, clusterName)
	if err != nil {
		return nil, err
	}

	return clientcmd.Write(*kubeconfig)
}

func buildNewKubeconfig(ca *triple.KeyPair, server, commonName string, organizations []string, clusterName string) (*clientcmdapi.Config, error) {
	baseKubconfig := getBaseKubeconfig(ca.Cert, server, clusterName)

	kp, err := triple.NewClientKeyPair(ca, commonName, organizations)
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %v", err)
	}

	baseKubconfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		KubeconfigDefaultContextKey: {
			ClientCertificateData: certutil.EncodeCertPEM(kp.Cert),
			ClientKeyData:         certutil.EncodePrivateKeyPEM(kp.Key),
		},
	}

	return baseKubconfig, nil
}

func getBaseKubeconfig(caCert *x509.Certificate, server, clusterName string) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			// We use the actual cluster name here. It is later used in encodeKubeconfig()
			// to set the filename of the kubeconfig downloaded from API to `kubeconfig-clusterName`.
			clusterName: {
				CertificateAuthorityData: certutil.EncodeCertPEM(caCert),
				Server: server,
			},
		},
		CurrentContext: KubeconfigDefaultContextKey,
		Contexts: map[string]*clientcmdapi.Context{
			KubeconfigDefaultContextKey: {
				Cluster:  clusterName,
				AuthInfo: KubeconfigDefaultContextKey,
			},
		},
	}
}

func isValidKubeconfig(kubeconfigBytes []byte, caCert *x509.Certificate, server, commonName, clusterName string) (bool, error) {
	if len(kubeconfigBytes) == 0 {
		return false, nil
	}

	existingKubeconfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return false, err
	}

	baseKubeconfig := getBaseKubeconfig(caCert, server, clusterName)

	authInfo := existingKubeconfig.AuthInfos[KubeconfigDefaultContextKey]
	if authInfo == nil {
		return false, nil
	}

	// We set the AuthInfo to nil, to have something to compare easily.
	// As the AuthInfo contains the client cert, which will always be different
	existingKubeconfig.AuthInfos = nil
	if !equality.Semantic.DeepEqual(baseKubeconfig, existingKubeconfig) {
		return false, nil
	}

	// Now check if the client cert from the kubeconfig is still valid
	certs, err := certutil.ParseCertsPEM(authInfo.ClientCertificateData)
	if err != nil {
		return false, err
	}

	if !IsClientCertificateValidForAllOf(certs[0], commonName, nil) {
		return false, nil
	}

	return true, nil
}
