/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"crypto/x509"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

type adminKubeconfigCreatorData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// AdminKubeconfigCreator returns a function to create/update the secret with the admin kubeconfig
func AdminKubeconfigCreator(data adminKubeconfigCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return AdminKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			config := GetBaseKubeconfig(ca.Cert, data.Cluster().Address.URL, data.Cluster().Name)
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				KubeconfigDefaultContextKey: {
					Token: data.Cluster().Address.AdminToken,
				},
			}

			b, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			se.Data[KubeconfigSecretKey] = b

			return se, nil
		}
	}
}

// ViewerKubeconfigCreator returns a function to create/update the secret with the viewer kubeconfig
func ViewerKubeconfigCreator(data *TemplateData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return ViewerKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			config := GetBaseKubeconfig(ca.Cert, data.Cluster().Address.URL, data.Cluster().Name)
			token, err := data.GetViewerToken()
			if err != nil {
				return nil, fmt.Errorf("failed to get token: %v", err)
			}
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				KubeconfigDefaultContextKey: {
					Token: token,
				},
			}

			b, err := clientcmd.Write(*config)
			if err != nil {
				return nil, err
			}

			se.Data[KubeconfigSecretKey] = b

			return se, nil
		}
	}
}

type internalKubeconfigCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

// GetInternalKubeconfigCreator is a generic function to return a secret generator to create a kubeconfig which must only be used within the seed-cluster as it uses the ClusterIP of the apiserver.
func GetInternalKubeconfigCreator(name, commonName string, organizations []string, data internalKubeconfigCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return name, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %v", err)
			}

			b := se.Data[KubeconfigSecretKey]
			apiserverURL := fmt.Sprintf("https://%s:%d", data.Cluster().Address.InternalName, data.Cluster().Address.Port)
			valid, err := IsValidKubeconfig(b, ca.Cert, apiserverURL, commonName, organizations, data.Cluster().Name)
			if err != nil || !valid {
				if err != nil {
					klog.V(2).Infof("failed to validate existing kubeconfig from %s/%s %v. Regenerating it...", se.Namespace, se.Name, err)
				} else {
					klog.V(2).Infof("invalid/outdated kubeconfig found in %s/%s. Regenerating it...", se.Namespace, se.Name)
				}

				se.Data[KubeconfigSecretKey], err = BuildNewKubeconfigAsByte(ca, apiserverURL, commonName, organizations, data.Cluster().Name)
				if err != nil {
					return nil, fmt.Errorf("failed to create new kubeconfig: %v", err)
				}
				return se, nil
			}

			return se, nil
		}
	}
}

func BuildNewKubeconfigAsByte(ca *triple.KeyPair, server, commonName string, organizations []string, clusterName string) ([]byte, error) {
	kubeconfig, err := buildNewKubeconfig(ca, server, commonName, organizations, clusterName)
	if err != nil {
		return nil, err
	}

	return clientcmd.Write(*kubeconfig)
}

func buildNewKubeconfig(ca *triple.KeyPair, server, commonName string, organizations []string, clusterName string) (*clientcmdapi.Config, error) {
	baseKubconfig := GetBaseKubeconfig(ca.Cert, server, clusterName)

	kp, err := triple.NewClientKeyPair(ca, commonName, organizations)
	if err != nil {
		return nil, fmt.Errorf("failed to create key pair: %v", err)
	}

	baseKubconfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		KubeconfigDefaultContextKey: {
			ClientCertificateData: triple.EncodeCertPEM(kp.Cert),
			ClientKeyData:         triple.EncodePrivateKeyPEM(kp.Key),
		},
	}

	return baseKubconfig, nil
}

func GetBaseKubeconfig(caCert *x509.Certificate, server, clusterName string) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			// We use the actual cluster name here. It is later used in encodeKubeconfig()
			// to set the filename of the kubeconfig downloaded from API to `kubeconfig-clusterName`.
			clusterName: {
				CertificateAuthorityData: triple.EncodeCertPEM(caCert),
				Server:                   server,
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

func IsValidKubeconfig(kubeconfigBytes []byte, caCert *x509.Certificate, server, commonName string, organizations []string, clusterName string) (bool, error) {
	if len(kubeconfigBytes) == 0 {
		return false, nil
	}

	existingKubeconfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return false, err
	}

	baseKubeconfig := GetBaseKubeconfig(caCert, server, clusterName)

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

	if !IsClientCertificateValidForAllOf(certs[0], commonName, organizations, caCert) {
		return false, nil
	}

	return true, nil
}
