package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"

	corev1 "k8s.io/api/core/v1"
)

// KubeletClientCertificate returns a secret with the etcd client certificate
func KubeletClientCertificate(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	return certificates.GetClientCertificateCreator(
		resources.KubeletClientCertificatesSecretName,
		"kube-apiserver-kubelet-client",
		[]string{"system:masters"},
		resources.KubeletClientCertSecretKey,
		resources.KubeletClientKeySecretKey,
		data.GetRootCA)(data, existing)
}
