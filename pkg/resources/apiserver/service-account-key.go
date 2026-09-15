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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type serviceAccountKeyReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
}

// ServiceAccountKeyReconciler returns a function to create/update a secret with the ServiceAccount key.
func ServiceAccountKeyReconciler(data serviceAccountKeyReconcilerData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.ServiceAccountKeySecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			// The key is written exactly once. Rotating it would invalidate every
			// service account token in the user cluster, so an existing secret is
			// never touched, whatever the cluster's key configuration says.
			if _, exists := se.Data[resources.ServiceAccountKeySecretKey]; exists {
				return se, nil
			}

			keyConfig, err := certificates.ServiceAccountKeyConfig(data.Cluster())
			if err != nil {
				return nil, err
			}

			priv, err := keyConfig.GenerateKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate %s service account key: %w", keyConfig, err)
			}

			privKeyPEM, err := triple.MarshalPrivateKeyPEM(priv)
			if err != nil {
				return nil, err
			}

			publicKeyPEM, err := triple.MarshalPublicKeyPEM(priv)
			if err != nil {
				return nil, err
			}

			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data[resources.ServiceAccountKeySecretKey] = privKeyPEM
			se.Data[resources.ServiceAccountKeyPublicKey] = publicKeyPEM
			return se, nil
		}
	}
}
