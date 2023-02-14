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
	"context"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/validation"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = appskubermaticv1.AddToScheme(testScheme)
}

func TestDefaultApplicationInstallation(t *testing.T) {
	appDef := &appskubermaticv1.ApplicationDefinition{ObjectMeta: metav1.ObjectMeta{Name: "appDef-1"}, Spec: appskubermaticv1.ApplicationDefinitionSpec{Description: "Description", Versions: []appskubermaticv1.ApplicationVersion{{Version: "v1.0.0"}}}}
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(appDef).
		Build()

	tests := []struct {
		name        string
		appInstall  *appskubermaticv1.ApplicationInstallation
		expectedApp *appskubermaticv1.ApplicationInstallation
		wantErr     bool
	}{
		{
			name:        "test no mutation - deployOpts is nil",
			appInstall:  getApplicationInstallation(nil),
			expectedApp: getApplicationInstallation(nil),
			wantErr:     false,
		},
		{
			name:        "test no mutation - deployOpts.Helm is nil",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{}),
			wantErr:     false,
		},
		{
			name:        "test no mutation - deployOpts.Helm is all filled",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			wantErr:     false,
		},
		{
			name:        "test no mutation - deployOpts.Helm is all filled (atomic=false)",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: 1}}}),
			wantErr:     false,
		},
		{
			name:        "test no mutation - deployOpts.Helm is all filled (atomic=false, wait= false, timeout = 0)",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: false, Timeout: metav1.Duration{Duration: 0}}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: false, Timeout: metav1.Duration{Duration: 0}}}),
			wantErr:     false,
		},
		{
			name:        "test mutation - deployOpts.Helm atomic =true --> wait and timeout should be defaulted",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:     false,
		},
		{
			name:        "test mutation - deployOpts.Helm atomic =true  and wait= true -->  timeout should be defaulted",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:     false,
		},
		{
			name:        "test mutation - deployOpts.Helm atomic =false  and wait= true -->  timeout should be defaulted",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: false, Wait: true, Timeout: metav1.Duration{Duration: defaulting.DefaultHelmTimeout}}}),
			wantErr:     false,
		},
		{
			name:        "test mutation - deployOpts.Helm atomic =true and timeout=10  --> wait should be defaulted",
			appInstall:  getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Timeout: metav1.Duration{Duration: 10}}}),
			expectedApp: getApplicationInstallation(&appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{Atomic: true, Wait: true, Timeout: metav1.Duration{Duration: 10}}}),
			wantErr:     false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := defaulting.DefaultApplicationInstallation(tc.appInstall)
			if tc.wantErr != (err != nil) {
				t.Fatalf("DefaultApplicationInstallation() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !diff.SemanticallyEqual(tc.appInstall, tc.expectedApp) {
				t.Fatalf("mutate applicationInstllation differs from expected:\n%s", diff.ObjectDiff(tc.expectedApp, tc.appInstall))
			}

			// test that mutate object is valid
			if errs := validation.ValidateApplicationInstallationSpec(context.Background(), fakeClient, *tc.appInstall); len(errs) > 0 {
				t.Fatalf("mutated applicationInstllation does not pass validation: %v", errs)
			}
		})
	}
}

func getApplicationInstallation(deployOpts *appskubermaticv1.DeployOptions) *appskubermaticv1.ApplicationInstallation {
	return &appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-1",
			Namespace: "kube-system",
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name: "default",
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    "appDef-1",
				Version: "v1.0.0",
			},
			DeployOptions: deployOpts,
		},
	}
}
