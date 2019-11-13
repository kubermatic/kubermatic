package openshift

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"

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

const OAuthBootstrapSecretName = "kubeadmin"

// OAuthBootstrapPassword is the password we use to authenticate the dashboard against the OAuth
// service. It must be created in the kube-system namespace.
func OAuthBootstrapPassword() (string, reconciling.SecretCreator) {
	return OAuthBootstrapSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {

		if s.Data == nil {
			s.Data = map[string][]byte{}
		}
		// The only way this ever gets updated is if someone empties it. It won't be accepted if
		// its creation timestamp is after kube-system namespace creation timestamp + 1h.
		// https://github.com/openshift/origin/blob/e774f85c15aef11d76db1ffc458484867e503293/pkg/oauthserver/authenticator/password/bootstrap/bootstrap.go#L131
		if _, exists := s.Data[OAuthBootstrapSecretName]; exists {
			return s, nil
		}

		rawPassword, err := generateNewOAuthSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %v", err)
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %v", err)
		}
		s.Data[OAuthBootstrapSecretName] = hashedPassword

		return s, nil
	}
}

func generateNewOAuthSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read from crypto/rand: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
