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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
)

var (
	applicationNamespace = appskubermaticv1.AppNamespaceSpec{
		Name:   "test-namespace",
		Create: true,
	}
)

func TestGetAppNamespace(t *testing.T) {
	testCases := []struct {
		name        string
		application appskubermaticv1.ApplicationDefinition
		validate    func(applications appskubermaticv1.ApplicationDefinition) bool
	}{
		{
			name:        "scenario 1: application namespace should be set to default value when a default value is configured",
			application: *genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue, nil, &applicationNamespace),
			validate: func(application appskubermaticv1.ApplicationDefinition) bool {
				namespace := getAppNamespace(&application)
				return namespace.Name == applicationNamespace.Name
			},
		},
		{
			name:        "scenario 2: application namespace should be set to application name when no default value is configured",
			application: *genApplicationDefinition("applicationName", "namespace", "v1.0.0", "", true, false, defaultValue, nil, nil),
			validate: func(application appskubermaticv1.ApplicationDefinition) bool {
				namespace := getAppNamespace(&application)
				return namespace.Name == application.Name
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// validate the result
			if !test.validate(test.application) {
				t.Fatalf("Validation failed for %v", test.name)
			}
		})
	}
}
