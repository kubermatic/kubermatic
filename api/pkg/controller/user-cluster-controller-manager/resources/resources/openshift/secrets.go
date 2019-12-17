package openshift

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/servingcerthelper"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

const (
	OAuthBootstrapSecretName       = "kubeadmin"
	OAuthBootstrapEncryptedkeyName = "encrypted"
)

// OAuthBootstrapPassword is the password we use to authenticate the dashboard against the OAuth
// service. It must be created in the kube-system namespace.
// We also have to transport its raw value into the seed, because its used by the Openshift Console endpoint
// to authenticate against the oauth service. To not expose the raw value to the user, we AES encrypt it using the
// admin token as key (Anyone with that token may do everything in the seed anyways).
func OAuthBootstrapPasswordCreatorGetter(seedClient ctrlruntimeclient.Client, seedNamespace string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return OAuthBootstrapSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {

			adminTokenSecret := &corev1.Secret{}
			name := types.NamespacedName{
				Namespace: seedNamespace,
				Name:      resources.AdminKubeconfigSecretName,
			}
			if err := seedClient.Get(context.Background(), name, adminTokenSecret); err != nil {
				return nil, fmt.Errorf("failed to get admin token secret: %v", err)
			}
			adminTokenSecretValue, adminTokenSecretValueExists := adminTokenSecret.Data["token"]
			if !adminTokenSecretValueExists {
				return nil, errors.New("adminTokenSecret has no `token` key")
			}

			// The only way this ever gets updated is if someone empties it. It won't be accepted if
			// its creation timestamp is after kube-system namespace creation timestamp + 1h.
			// https://github.com/openshift/origin/blob/e774f85c15aef11d76db1ffc458484867e503293/pkg/oauthserver/authenticator/password/bootstrap/bootstrap.go#L131
			if encryptedValue, exists := s.Data[OAuthBootstrapEncryptedkeyName]; exists {
				plainValue, err := AESDecrypt(encryptedValue, adminTokenSecretValue)
				if err != nil {
					// The openshift seed sync controller will empty the secret if this can not be encrypted, so just return here
					return nil, fmt.Errorf("failed to decrypt: %v", err)
				}
				hashedValue, err := bcrypt.GenerateFromPassword(plainValue, 12)
				if err != nil {
					return nil, fmt.Errorf("failed to hash existing password: %v", err)
				}
				if string(s.Data[OAuthBootstrapSecretName]) == string(hashedValue) {
					return s, nil
				}
			}

			if s.Data == nil {
				s.Data = map[string][]byte{}
			}

			rawPassword, err := generateNewSecret()
			if err != nil {
				return nil, fmt.Errorf("failed to generate password: %v", err)
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
			if err != nil {
				return nil, fmt.Errorf("failed to hash password: %v", err)
			}
			encyptedPassword, err := AESEncrypt([]byte(rawPassword), adminTokenSecretValue)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt password: %v", err)
			}
			s.Data[OAuthBootstrapSecretName] = hashedPassword
			// We need this to be available in the seed so our API can use it to authenticate against the Oauth service
			s.Data[OAuthBootstrapEncryptedkeyName] = encyptedPassword

			return s, nil
		}
	}
}

// Based on https://golang.org/src/crypto/cipher/example_test.go
func AESEncrypt(data, key []byte) ([]byte, error) {
	if n := len(key); n < 16 {
		return nil, fmt.Errorf("key must at least be 16 bytes long, was %d", n)
	}
	block, err := aes.NewCipher(key[0:16])
	if err != nil {
		return nil, fmt.Errorf("failed to construct cipher: %v", err)
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to gather entropy: %v", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to construct aesgcm: %v", err)
	}
	return append(nonce, aesgcm.Seal(nil, nonce, data, nil)...), nil
}

func AESDecrypt(data, key []byte) ([]byte, error) {
	if n := len(key); n < 16 {
		return nil, fmt.Errorf("key must at least be 16 bytes long, was %d", n)
	}
	if n := len(data); n < 12 {
		return nil, fmt.Errorf("data must at least be 12 bytes, got %d", n)
	}
	block, err := aes.NewCipher(key[0:16])
	if err != nil {
		return nil, fmt.Errorf("failed to construct cipher: %v", err)
	}

	nonce := data[:12]
	data = data[12:]

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to construct aesgcm: %v", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	return plaintext, nil
}

func generateNewSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read from crypto/rand: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
