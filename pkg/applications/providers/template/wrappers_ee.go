//go:build ee

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	eeutil "k8c.io/kubermatic/v2/pkg/ee/applications"
)

func (h *HelmTemplate) templatePreDefinedValues(applicationValues map[string]any) (map[string]any, error) {
	templateData, err := eeutil.GetTemplateData(h.Ctx, h.SeedClient, h.ClusterName)

	if err != nil {
		return nil, err
	}
	return eeutil.RenderValueTemplate(applicationValues, templateData)
}
