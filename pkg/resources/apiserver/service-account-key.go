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

package apiserver

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountKeyCreator returns a function to create/update a secret with the ServiceAccount key
func ServiceAccountKeyCreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.ServiceAccountKeySecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if _, exists := se.Data[resources.ServiceAccountKeySecretKey]; exists {
				return se, nil
			}
			priv, err := rsa.GenerateKey(cryptorand.Reader, 2048)
			if err != nil {
				return nil, err
			}
			saKey := x509.MarshalPKCS1PrivateKey(priv)
			privKeyBlock := pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: saKey,
			}
			publicKeyDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
			if err != nil {
				return nil, err
			}
			publicKeyBlock := pem.Block{
				Type:    "PUBLIC KEY",
				Headers: nil,
				Bytes:   publicKeyDer,
			}
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data[resources.ServiceAccountKeySecretKey] = pem.EncodeToMemory(&privKeyBlock)
			se.Data[resources.ServiceAccountKeyPublicKey] = pem.EncodeToMemory(&publicKeyBlock)
			return se, nil

		}
	}

}
