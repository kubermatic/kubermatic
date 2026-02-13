/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package applicationcatalogmanager

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
)

// getImage constructs the container image reference from configuration,
// falling back to defaults if not specified.
func getImage(cfg *kubermaticv1.KubermaticConfiguration) string {
	repository := defaulting.DefaultApplicationManagerImageRepository
	if cfg.Spec.Applications.CatalogManager.Image.Repository != "" {
		repository = cfg.Spec.Applications.CatalogManager.Image.Repository
	}

	tag := defaulting.DefaultApplicationManagerImageTag
	if cfg.Spec.Applications.CatalogManager.Image.Tag != "" {
		tag = cfg.Spec.Applications.CatalogManager.Image.Tag
	}

	return repository + ":" + tag
}
