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
	"bytes"
	"encoding/csv"

	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// TokenUsers returns a secret containing the tokens csv.
func TokenUsersReconciler(data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.TokensSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			buffer := &bytes.Buffer{}
			writer := csv.NewWriter(buffer)
			if err := writer.Write([]string{data.Cluster().Status.Address.AdminToken, "admin", "10000", "system:masters"}); err != nil {
				return nil, err
			}
			viewerToken, err := data.GetViewerToken()
			if err != nil {
				return nil, err
			}
			if err := writer.Write([]string{viewerToken, "viewer", "10001", "viewers"}); err != nil {
				return nil, err
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				return nil, err
			}

			se.Data[resources.TokensSecretKey] = buffer.Bytes()
			return se, nil
		}
	}
}

// TokenViewerReconciler returns a secret containing the viewer token.
func TokenViewerReconciler() reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.ViewerTokenSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			if _, ok := se.Data[resources.ViewerTokenSecretKey]; !ok {
				se.Data[resources.ViewerTokenSecretKey] = []byte(kubernetes.GenerateToken())
			}

			return se, nil
		}
	}
}
