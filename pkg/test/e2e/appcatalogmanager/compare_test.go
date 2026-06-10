//go:build e2e

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

package appcatalogmanager

import (
	"strings"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeAppDef(name string, mutators ...func(*appskubermaticv1.ApplicationDefinition)) appskubermaticv1.ApplicationDefinition {
	app := appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			DisplayName:        "Display " + name,
			Description:        "Description " + name,
			Method:             appskubermaticv1.HelmTemplateMethod,
			DocumentationURL:   "https://docs.example.com/" + name,
			SourceURL:          "https://github.com/example/" + name,
			Logo:               "bG9nbw==",
			LogoFormat:         "png",
			DefaultValuesBlock: "key: value\n",
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "oci://quay.io/kubermatic-mirror/helm-charts",
								ChartName:    name,
								ChartVersion: "1.0.0",
							},
						},
					},
				},
			},
		},
	}

	for _, mutate := range mutators {
		mutate(&app)
	}

	return app
}

func TestCompareApplicationDefinitionSpecs(t *testing.T) {
	testcases := []struct {
		name        string
		oldApps     []appskubermaticv1.ApplicationDefinition
		newApps     []appskubermaticv1.ApplicationDefinition
		wantErr     bool
		errContains string
	}{
		{
			name:    "identical apps are compatible",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
		},
		{
			name:    "extra apps in new are accepted",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit"), makeAppDef("metallb")},
		},
		{
			name:        "missing app in new is rejected",
			oldApps:     []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit"), makeAppDef("metallb")},
			newApps:     []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			wantErr:     true,
			errContains: `application "metallb" is missing`,
		},
		{
			name:    "cosmetic metadata divergence is accepted",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.DisplayName = "Other display name"
					app.Spec.Description = "Other description"
					app.Spec.DocumentationURL = "https://sozercan.github.io/aikit/docs"
					app.Spec.SourceURL = "https://github.com/sozercan/aikit"
					app.Spec.Logo = "b3RoZXItbG9nbw=="
				}),
			},
		},
		{
			name:    "extra versions in new are accepted",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					extra := app.Spec.Versions[0].DeepCopy()
					extra.Version = "2.0.0"
					extra.Template.Source.Helm.ChartVersion = "2.0.0"
					app.Spec.Versions = append(app.Spec.Versions, *extra)
				}),
			},
		},
		{
			name: "missing version in new is rejected",
			oldApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					extra := app.Spec.Versions[0].DeepCopy()
					extra.Version = "2.0.0"
					app.Spec.Versions = append(app.Spec.Versions, *extra)
				}),
			},
			newApps:     []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			wantErr:     true,
			errContains: `version "2.0.0" is missing`,
		},
		{
			name:    "changed template for shared version is rejected",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.Versions[0].Template.Source.Helm.URL = "oci://other.registry/charts"
				}),
			},
			wantErr:     true,
			errContains: `version "1.0.0" differs`,
		},
		{
			name:    "changed default values block is rejected",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.DefaultValuesBlock = "key: other\n"
				}),
			},
			wantErr:     true,
			errContains: "behavior-relevant spec fields differ",
		},
		{
			name:    "changed default flag is rejected",
			oldApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.Default = true
				}),
			},
			wantErr:     true,
			errContains: "behavior-relevant spec fields differ",
		},
		{
			name: "explicit false bool pointers equal nil pointers",
			oldApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					falseVal := false
					app.Spec.Versions[0].Template.Source.Helm.Insecure = &falseVal
					app.Spec.Versions[0].Template.Source.Helm.PlainHTTP = &falseVal
				}),
			},
			newApps: []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
		},
		{
			name: "explicit true bool pointer differing from nil is rejected",
			oldApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					trueVal := true
					app.Spec.Versions[0].Template.Source.Helm.Insecure = &trueVal
				}),
			},
			newApps:     []appskubermaticv1.ApplicationDefinition{makeAppDef("aikit")},
			wantErr:     true,
			errContains: `version "1.0.0" differs`,
		},
		{
			name: "selector datacenter order is ignored",
			oldApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.Selector.Datacenters = []string{"dc-b", "dc-a"}
				}),
			},
			newApps: []appskubermaticv1.ApplicationDefinition{
				makeAppDef("aikit", func(app *appskubermaticv1.ApplicationDefinition) {
					app.Spec.Selector.Datacenters = []string{"dc-a", "dc-b"}
				}),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := compareApplicationDefinitionSpecs(tc.oldApps, tc.newApps)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error to contain %q, got: %v", tc.errContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
