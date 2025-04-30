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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
)

const (
	// kubeconfigDefaultAuthInfoKey is the Auth Info key used for all kubeconfigs.
	kubeconfigDefaultAuthInfoKey = "default"
)

type adminKubeconfigReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	GetRootCA() (*triple.KeyPair, error)
}

// AdminKubeconfigReconciler returns a function to create/update the secret with the admin kubeconfig.
func AdminKubeconfigReconciler(data adminKubeconfigReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return AdminKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %w", err)
			}

			address := data.Cluster().Status.Address
			config := GetBaseKubeconfig(ca.Cert, address.URL, data.Cluster().Name)
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				kubeconfigDefaultAuthInfoKey: {
					Token: address.AdminToken,
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

// ViewerKubeconfigReconciler returns a function to create/update the secret with the viewer kubeconfig.
func ViewerKubeconfigReconciler(data *TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return ViewerKubeconfigSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %w", err)
			}

			config := GetBaseKubeconfig(ca.Cert, data.Cluster().Status.Address.URL, data.Cluster().Name)
			token, err := data.GetViewerToken()
			if err != nil {
				return nil, fmt.Errorf("failed to get token: %w", err)
			}
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				kubeconfigDefaultAuthInfoKey: {
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

type internalKubeconfigReconcilerData interface {
	GetRootCA() (*triple.KeyPair, error)
	Cluster() *kubermaticv1.Cluster
}

// GetInternalKubeconfigReconciler is a generic function to return a secret generator to create a kubeconfig which must only be used within the seed-cluster as it uses the ClusterIP of the apiserver.
func GetInternalKubeconfigReconciler(namespace, name, commonName string, organizations []string, data internalKubeconfigReconcilerData, log *zap.SugaredLogger) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return name, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster ca: %w", err)
			}

			b := se.Data[KubeconfigSecretKey]
			apiserverURL := fmt.Sprintf("https://%s", data.Cluster().Status.Address.InternalName)
			valid, err := IsValidKubeconfig(b, ca.Cert, apiserverURL, commonName, organizations, data.Cluster().Name)
			if err != nil || !valid {
				objLogger := log.With("namespace", namespace, "name", name)
				if err != nil {
					objLogger.Infow("failed to validate existing kubeconfig, regenerating", zap.Error(err))
				} else {
					objLogger.Info("invalid/outdated kubeconfig found, regenerating")
				}

				se.Data[KubeconfigSecretKey], err = BuildNewKubeconfigAsByte(ca, apiserverURL, commonName, organizations, data.Cluster().Name)
				if err != nil {
					return nil, fmt.Errorf("failed to create new kubeconfig: %w", err)
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
		return nil, fmt.Errorf("failed to create key pair: %w", err)
	}

	baseKubconfig.AuthInfos = map[string]*clientcmdapi.AuthInfo{
		kubeconfigDefaultAuthInfoKey: {
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
		CurrentContext: clusterName,
		Contexts: map[string]*clientcmdapi.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: kubeconfigDefaultAuthInfoKey,
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

	authInfo := existingKubeconfig.AuthInfos[kubeconfigDefaultAuthInfoKey]
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
