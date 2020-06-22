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

package resources

import (
	"errors"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const openshiftImagePullSecretName = "openshift-image-pull-secret"

func ImagePullSecretCreator(cluster *kubermaticv1.Cluster) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return openshiftImagePullSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson
			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			// Should never happen
			if cluster.Spec.Openshift == nil {
				return nil, errors.New("openshift spec is nil")
			}
			s.Data[corev1.DockerConfigJsonKey] = []byte(cluster.Spec.Openshift.ImagePullSecret)
			return s, nil
		}
	}
}
