package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
)

type kubeletClientCertificateCreatorData interface {
	GetRootCA() (*triple.KeyPair, error)
}

// KubeletClientCertificateCreator returns a function to create/update a secret with the client certificate for the apiserver -> kubelet connection.
func KubeletClientCertificateCreator(data kubeletClientCertificateCreatorData) resources.NamedSecretCreatorGetter {
	return certificates.GetClientCertificateCreator(
		resources.KubeletClientCertificatesSecretName,
		"kube-apiserver-kubelet-client",
		[]string{"system:masters"},
		resources.KubeletClientCertSecretKey,
		resources.KubeletClientKeySecretKey,
		data.GetRootCA,
	)
}
