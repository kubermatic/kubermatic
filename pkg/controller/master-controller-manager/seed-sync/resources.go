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

package seedsync

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

func secretCreator(original *corev1.Secret) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return original.Name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Labels = original.Labels
			if s.Labels == nil {
				s.Labels = make(map[string]string)
			}
			s.Labels[ManagedByLabel] = ControllerName

			s.Annotations = original.Annotations
			s.Data = original.Data

			return s, nil
		}
	}
}

func seedCreator(seed *kubermaticv1.Seed) reconciling.NamedSeedCreatorGetter {
	return func() (string, reconciling.SeedCreator) {
		return seed.Name, func(s *kubermaticv1.Seed) (*kubermaticv1.Seed, error) {
			s.Labels = seed.Labels
			if s.Labels == nil {
				s.Labels = make(map[string]string)
			}
			s.Labels[ManagedByLabel] = ControllerName

			s.Annotations = seed.Annotations
			s.Spec = seed.Spec

			return s, nil
		}
	}
}

func configCreator(config *kubermaticv1.KubermaticConfiguration) reconciling.NamedKubermaticConfigurationCreatorGetter {
	return func() (string, reconciling.KubermaticConfigurationCreator) {
		return config.Name, func(c *kubermaticv1.KubermaticConfiguration) (*kubermaticv1.KubermaticConfiguration, error) {
			c.Labels = config.Labels
			if c.Labels == nil {
				c.Labels = make(map[string]string)
			}
			c.Labels[ManagedByLabel] = ControllerName

			c.Annotations = config.Annotations
			c.Spec = config.Spec

			return c, nil
		}
	}
}
