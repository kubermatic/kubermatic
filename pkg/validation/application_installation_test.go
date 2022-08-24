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

package validation

import (
	"context"
	"fmt"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	defaultAppName  = "app"
	invalidResource = "invalid"
)

var (
	testScheme                 = runtime.NewScheme()
	defaultAppVersion          = appskubermaticv1.Version{Version: *semverlib.MustParse("1.2.3")}
	defaultAppSecondaryVersion = appskubermaticv1.Version{Version: *semverlib.MustParse("1.2.4")}
)

func init() {
	_ = appskubermaticv1.AddToScheme(testScheme)
}

// TestValidateApplicationInstallationSpec tests the validation for ApplicationInstallation creation.
func TestValidateApplicationInstallationSpec(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName)
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(ad).
		Build()

	ai := getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion)

	testCases := []struct {
		name          string
		ai            *appskubermaticv1.ApplicationInstallation
		expectedError string
	}{
		{
			name:          "Create ApplicationInstallation Success",
			ai:            ai,
			expectedError: "[]",
		},
		{
			name: "Create ApplicationInstallation Failure - ApplicationDefinitation doesn't exist",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Name = invalidResource
					return *spec
				}(),
			}, expectedError: `[spec.applicationRef.name: Not found: "invalid"]`,
		},
		{
			name: "Create ApplicationInstallation Failure - Invalid Version",
			ai: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Version = appskubermaticv1.Version{Version: *semverlib.MustParse("3.2.3")}
					return *spec
				}(),
			}, expectedError: `[spec.applicationRef.version: Not found: 3.2.3]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationInstallationSpec(context.Background(), fakeClient, testCase.ai.Spec)
			if fmt.Sprint(err) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}

// TestValidateApplicationInstallationSpec tests the validation for ApplicationInstallation creation.
func TestValidateApplicationInstallationUpdate(t *testing.T) {
	ad := getApplicationDefinition(defaultAppName)
	updatedAD := getApplicationDefinition("updated-app")
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(ad, updatedAD).
		Build()

	ai := getApplicationInstallation(defaultAppName, defaultAppName, defaultAppVersion)

	testCases := []struct {
		name          string
		ai            *appskubermaticv1.ApplicationInstallation
		updatedAI     *appskubermaticv1.ApplicationInstallation
		expectedError string
	}{
		{
			name: "Update ApplicationInstallation Success",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Labels = map[string]string{"key": "value"}
					spec.ApplicationRef.Version = defaultAppSecondaryVersion
					return *spec
				}(),
			},
			expectedError: "[]",
		},
		{
			name: "Update ApplicationInstallation Failure - .Namespace.Name is immutable",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.Namespace.Name = invalidResource
					return *spec
				}(),
			},
			expectedError: `[spec.namespace.name: Invalid value: "default": field is immutable]`,
		},
		{
			name: "Update ApplicationInstallation Failure - .ApplicationRef.Name is immutable",
			ai:   ai,
			updatedAI: &appskubermaticv1.ApplicationInstallation{
				Spec: func() appskubermaticv1.ApplicationInstallationSpec {
					spec := ai.Spec.DeepCopy()
					spec.ApplicationRef.Name = "updated-app"
					return *spec
				}(),
			},
			expectedError: `[spec.applicationRef.name: Invalid value: "app": field is immutable]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationInstallationUpdate(context.Background(), fakeClient, *testCase.ai, *testCase.updatedAI)
			if fmt.Sprint(err) != testCase.expectedError {
				if testCase.expectedError == "[]" {
					testCase.expectedError = "nil"
				}
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}

func getApplicationDefinition(name string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "Description",
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: defaultAppVersion,
				},
				{
					Version: defaultAppSecondaryVersion,
				},
			},
		},
	}
}

func getApplicationInstallation(name string, appName string, appVersion appskubermaticv1.Version) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kube-system",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.NamespaceSpec{
				Name:   "default",
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    appName,
				Version: appVersion,
			},
		},
	}
}
