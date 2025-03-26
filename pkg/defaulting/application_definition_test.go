/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package defaulting_test

import (
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/validation"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultApplicationDefinition(t *testing.T) {
	tests := []struct {
		name           string
		appDef         *appskubermaticv1.ApplicationDefinition
		expectedAppDef *appskubermaticv1.ApplicationDefinition
		wantErr        bool
	}{
		{
			name:           "test no mutation - DefaultDeployOps is nil",
			appDef:         getApplicationDefinition(nil),
			expectedAppDef: getApplicationDefinition(nil),
			wantErr:        false,
		},
		{
			name:           "test no mutation - DefaultDeployOps.Helm is nil",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{}),
			wantErr:        false,
		},
		{
			name:           "test no mutation - DefaultDeployOps.Helm is all filled",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			wantErr:        false,
		},
		{
			name:           "test no mutation - DefaultDeployOps.Helm is all filled (atomic=false)",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			wantErr:        false,
		},
		{
			name:           "test no mutation - DefaultDeployOps.Helm is all filled (atomic=false, wait= false, timeout = 0)",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: false, Timeout: metav1.Duration{Duration: 0}}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: false, Timeout: metav1.Duration{Duration: 0}}}),
			wantErr:        false,
		},
		{
			name:           "test mutation - DefaultDeployOps.Helm atomic =true --> wait and timeout should be defaulted",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:        false,
		},
		{
			name:           "test mutation - DefaultDeployOps.Helm atomic =true  and wait= true -->  timeout should be defaulted",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:        false,
		},
		{
			name:           "test mutation - DefaultDeployOps.Helm atomic =false  and wait= true -->  timeout should be defaulted",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:        false,
		},
		{
			name:           "test mutation - DefaultDeployOps.Helm atomic =true and timeout=10  --> wait should be defaulted",
			appDef:         getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Timeout: metav1.Duration{Duration: 10}}}),
			expectedAppDef: getApplicationDefinition(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 10}}}),
			wantErr:        false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := defaulting.DefaultApplicationDefinition(tc.appDef)
			if tc.wantErr != (err != nil) {
				t.Fatalf("defaultApplicationDefinition() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !diff.SemanticallyEqual(tc.appDef, tc.expectedAppDef) {
				t.Fatalf("mutate applicationDefinition differs from expected:\n%s", diff.ObjectDiff(tc.expectedAppDef, tc.appDef))
			}

			// test that mutate object is valid
			if errs := validation.ValidateApplicationDefinitionSpec(*tc.appDef); len(errs) > 0 {
				t.Fatalf("mutated applicationDefinition does not pass validation: %v", errs)
			}
		})
	}
}

func getApplicationDefinition(defaultDeployOps *appskubermaticv1.DeployOptions) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApplicationDefinition",
			APIVersion: appskubermaticv1.GroupName + "/" + appskubermaticv1.GroupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-1",
			Namespace: "kube-system",
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "an app",
			Method:      appskubermaticv1.HelmTemplateMethod,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://example.com/some-repo",
								ChartName:    "chartName",
								ChartVersion: "1.0.0",
							},
						},
					},
				},
			},
			DefaultDeployOptions: defaultDeployOps,
		},
	}
}
