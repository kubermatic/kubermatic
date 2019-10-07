package openshift

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func CloudCredentialSecretCreator(templateSecret corev1.Secret) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return templateSecret.Name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Data = templateSecret.Data
			return s, nil
		}
	}
}

func RegistryServingCert(caCert *triple.KeyPair) reconciling.NamedSecretCreatorGetter {
	caGetter := func() (*triple.KeyPair, error) {
		return caCert, nil
	}
	return servingcerthelper.ServingCertSecretCreator(caGetter,
		"image-registry-tls",
		"image-registry.openshift-image-registry.svc",
		[]string{"image-registry.openshift-image-registry.svc", "image-registry.openshift-image-registry.svc.cluster.local"},
		nil)
}
