package helper

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	// JournaldMaxUse defines the maximum space that journalD logs can occupy.
	// https://www.freedesktop.org/software/systemd/man/journald.conf.html#SystemMaxUse=
	JournaldMaxUse = "5G"
)

func GetServerAddressFromKubeconfig(kubeconfig *clientcmdapi.Config) (string, error) {
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return strings.Replace(clusterConfig.Server, "https://", "", -1), nil
	}

	return "", fmt.Errorf("no server address found")

}

func GetCACert(kubeconfig *clientcmdapi.Config) (string, error) {
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return string(clusterConfig.CertificateAuthorityData), nil
	}

	return "", fmt.Errorf("no CACert found")
}

// GetKubeadmCACertHash returns a sha256sum of the Certificates RawSubjectPublicKeyInfo
func GetKubeadmCACertHash(kubeconfig *clientcmdapi.Config) (string, error) {
	cacert, err := GetCACert(kubeconfig)
	if err != nil {
		return "", err
	}
	// _ is not an error but the remaining bytes in case the
	// input to pem.Decode() contains more than one cert
	certBlock, _ := pem.Decode([]byte(cacert))
	if certBlock == nil {
		return "", fmt.Errorf("pem certificate is empty")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", fmt.Errorf("error parsing certificate: %v", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(cert.RawSubjectPublicKeyInfo)), nil
}

func GetTokenFromKubeconfig(kubeconfig *clientcmdapi.Config) (string, error) {
	if len(kubeconfig.AuthInfos) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one token, can not extract token")
	}

	for _, authInfo := range kubeconfig.AuthInfos {
		return string(authInfo.Token), nil
	}

	return "", fmt.Errorf("no token found in kubeconfig")
}

// StringifyKubeconfig marshals a kubeconfig to its text form
func StringifyKubeconfig(kubeconfig *clientcmdapi.Config) (string, error) {
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error writing kubeconfig: %v", err)
	}

	return string(kubeconfigBytes), nil
}
