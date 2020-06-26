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

package openshift

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
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

			secretKey, err := GetOAuthEncryptionKey(context.Background(), seedClient, seedNamespace)
			if err != nil {
				return nil, fmt.Errorf("failed to get oauth encryption key: %v", err)
			}

			// The only way this ever gets updated is if someone empties it. It won't be accepted if
			// its creation timestamp is after kube-system namespace creation timestamp + 1h.
			// https://github.com/openshift/origin/blob/e774f85c15aef11d76db1ffc458484867e503293/pkg/oauthserver/authenticator/password/bootstrap/bootstrap.go#L131
			if encryptedValue, exists := s.Data[OAuthBootstrapEncryptedkeyName]; exists {
				if encryptedValueMatchesBcryptHash(encryptedValue, secretKey, s.Data[OAuthBootstrapSecretName]) {
					return s, nil
				}
			}

			s.Data = map[string][]byte{}

			rawPassword, err := generateNewSecret()
			if err != nil {
				return nil, fmt.Errorf("failed to generate password: %v", err)
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
			if err != nil {
				return nil, fmt.Errorf("failed to hash password: %v", err)
			}
			encyptedPassword, err := aesEncrypt([]byte(rawPassword), secretKey)
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

// GetOAuthEncryptionKey fetches the key used to encrypt the OAuthBootstrapPassword in the usercluster.
// We simply use the UID of the CA secret, as it it should be very hard to guess.
func GetOAuthEncryptionKey(ctx context.Context, seedClient ctrlruntimeclient.Client, seedNamespace string) ([]byte, error) {
	caSecret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: seedNamespace,
		Name:      resources.CASecretName,
	}
	if err := seedClient.Get(ctx, name, caSecret); err != nil {
		return nil, fmt.Errorf("failed to get ca secret: %v", err)
	}
	return []byte(string(caSecret.UID)), nil
}

// Based on https://golang.org/src/crypto/cipher/example_test.go
func aesEncrypt(data, key []byte) ([]byte, error) {
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

func encryptedValueMatchesBcryptHash(encryptedValue, key, hash []byte) bool {
	plainValue, err := AESDecrypt(encryptedValue, key)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword(hash, plainValue) == nil
}

func generateNewSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read from crypto/rand: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
