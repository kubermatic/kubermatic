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

package defaultapplicationcontroller

import (
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
)

func getAppNamespace(application *appskubermaticv1.ApplicationDefinition) *appskubermaticv1.AppNamespaceSpec {
	if application.Spec.DefaultNamespace != nil {
		return application.Spec.DefaultNamespace
	}
	return &appskubermaticv1.AppNamespaceSpec{
		Name:   application.Name,
		Create: true,
	}
}
