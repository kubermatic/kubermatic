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
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateApplicationDefinition(t *testing.T) {
	tm := metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
	cs := appskubermaticv1.ApplicationConstraints{K8sVersion: ">1.0.0", KKPVersion: ">1.0.0"}
	helmv := appskubermaticv1.ApplicationVersion{Version: "v1", Constraints: cs, Template: appskubermaticv1.ApplicationTemplate{Method: "helm"}}
	gitv := appskubermaticv1.ApplicationVersion{Version: "v2", Constraints: cs, Template: appskubermaticv1.ApplicationTemplate{Method: "helm"}}
	spec := appskubermaticv1.ApplicationDefinitionSpec{Versions: []appskubermaticv1.ApplicationVersion{helmv, gitv}}

	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid source helm": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{URL: "kubermatic.io", ChartName: "test", ChartVersion: "1.0.0"}}
					return *s
				}(),
			},
			0,
		},
		"valid source git": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{}}
					return *s
				}(),
			},
			0,
		},
		"mixed sources": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = gitv.Template.Source
					s.Versions[1].Template.Source = helmv.Template.Source
					return *s
				}(),
			},
			0,
		},
		"invalid method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = "invalid"
					return *s
				}(),
			},
			1,
		},
		"valid method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = "helm"
					return *s
				}(),
			},
			0,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = tm
			errl := ValidateApplicationDefinition(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateApplicationVersions(t *testing.T) {
	tt := map[string]struct {
		vs        []appskubermaticv1.ApplicationVersion
		expErrLen int
	}{
		"duplicate version": {
			[]appskubermaticv1.ApplicationVersion{
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "1"}},
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "1"}},
			},
			1,
		},
		"invalid kkp version": {
			[]appskubermaticv1.ApplicationVersion{
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "not-semver"}},
			},
			1,
		},
		"invalid k8s version": {
			[]appskubermaticv1.ApplicationVersion{
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "not-semver", KKPVersion: "1"}},
			},
			1,
		},
	}
	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			errl := ValidateApplicationVersions(tc.vs, nil)
			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}
