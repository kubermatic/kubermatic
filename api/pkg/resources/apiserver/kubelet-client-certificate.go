package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// KubeletClientCertificateCreator returns a function to create/update a secret with the client certificate for the apiserver -> kubelet connection.
func KubeletClientCertificateCreator(data resources.SecretDataProvider) resources.SecretCreator {
	return certificates.GetClientCertificateCreator(
		"kube-apiserver-kubelet-client",
		[]string{"system:masters"},
		resources.KubeletClientCertSecretKey,
		resources.KubeletClientKeySecretKey,
		data.GetRootCA)
}
