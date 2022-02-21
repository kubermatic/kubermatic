/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package helper

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func AutomaticBackupEnabled(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) bool {
	return cfg.Spec.SeedController.BackupRestore.Enabled || seed.IsDefaultEtcdAutomaticBackupEnabled()
}

func EffectiveBackupStoreContainer(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) (*corev1.Container, error) {
	// a customized container is configured
	if cfg.Spec.SeedController.BackupStoreContainer != "" {
		return ContainerFromString(cfg.Spec.SeedController.BackupStoreContainer)
	}

	if cfg.Spec.SeedController.BackupRestore.Enabled || seed.IsDefaultEtcdAutomaticBackupEnabled() {
		return ContainerFromString(defaults.DefaultNewBackupStoreContainer)
	}

	// use the legacy default container
	return ContainerFromString(defaults.DefaultBackupStoreContainer)
}

func EffectiveBackupCleanupContainer(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) (*corev1.Container, error) {
	// a customized container is configured
	if cfg.Spec.SeedController.BackupCleanupContainer != "" {
		return ContainerFromString(cfg.Spec.SeedController.BackupCleanupContainer)
	}

	// the cleanup container is only used by the legacy backup controller, so there is no further decision needed
	return ContainerFromString(defaults.DefaultBackupCleanupContainer)
}

func EffectiveBackupDeleteContainer(cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) (*corev1.Container, error) {
	// a customized container is configured
	if cfg.Spec.SeedController.BackupDeleteContainer != "" {
		return ContainerFromString(cfg.Spec.SeedController.BackupDeleteContainer)
	}

	// the delete container is only used by the new backup/restore controllers, so there is no further decision needed
	return ContainerFromString(defaults.DefaultNewBackupDeleteContainer)
}

func ContainerFromString(containerSpec string) (*corev1.Container, error) {
	if len(strings.TrimSpace(containerSpec)) == 0 {
		return nil, nil
	}

	container := &corev1.Container{}
	manifestDecoder := yaml.NewYAMLToJSONDecoder(strings.NewReader(containerSpec))
	if err := manifestDecoder.Decode(container); err != nil {
		return nil, err
	}

	// Just because it's a valid corev1.Container does not mean
	// the APIServer will accept it, thus we do some additional
	// checks
	if container.Name == "" {
		return nil, fmt.Errorf("container must have a name")
	}
	if container.Image == "" {
		return nil, fmt.Errorf("container must have an image")
	}

	return container, nil
}
