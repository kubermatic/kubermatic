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
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
)

var (
	applicationNamespace = appskubermaticv1.AppNamespaceSpec{
		Name:   "test-namespace",
		Create: true,
	}
)

func TestGetAppNamespace(t *testing.T) {
	testCases := []struct {
		name              string
		applicationName   string
		appNamespace      *appskubermaticv1.AppNamespaceSpec
		expectedNamespace string
	}{
		{
			name:              "scenario 1: application namespace should be set to default value when a default value is configured",
			applicationName:   "applicationName",
			appNamespace:      &applicationNamespace,
			expectedNamespace: applicationNamespace.Name,
		},
		{
			name:              "scenario 2: application namespace should be set to application name when no default value is configured",
			applicationName:   "applicationName",
			appNamespace:      nil,
			expectedNamespace: applicationName,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			application := *genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue, nil, test.appNamespace)
			namespace := getAppNamespace(&application)
			// validate the result
			if namespace.Name != test.expectedNamespace {
				t.Fatalf("Validation failed. Expected namespace %q, got %q instead", test.expectedNamespace, namespace.Name)
			}
		})
	}
}
