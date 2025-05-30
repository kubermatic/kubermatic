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
	"fmt"
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

var (
	secretKeySelector = &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "git-cred"}, Key: "thekey"}
	helmv             = appskubermaticv1.ApplicationVersion{Version: "v1", Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSource()}}}
	gitv              = appskubermaticv1.ApplicationVersion{Version: "v2", Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Git: validGitSource()}}}
	spec              = appskubermaticv1.ApplicationDefinitionSpec{Method: appskubermaticv1.HelmTemplateMethod, Versions: []appskubermaticv1.ApplicationVersion{helmv, gitv}}
)

func validHelmSource() *appskubermaticv1.HelmSource {
	return &appskubermaticv1.HelmSource{
		URL:          "http://example.com/charts",
		ChartName:    "apache",
		ChartVersion: "9.1.3",
		Credentials:  nil,
	}
}

func validGitSource() *appskubermaticv1.GitSource {
	return &appskubermaticv1.GitSource{
		Remote: "https://example.com/repo.git",
		Ref: appskubermaticv1.GitReference{
			Branch: "main",
		},
		Path:        "/",
		Credentials: nil,
	}
}

func TestValidateApplicationDefinitionSpec(t *testing.T) {
	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid source helm": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{URL: "https://kubermatic.io", ChartName: "test", ChartVersion: "1.0.0"}}
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
		"valid DeployOpts helm is nil": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{}
					return *s
				}(),
			},
			0,
		},
		"valid DeployOpts helm={wait: true, timeout: 5, atomic: true}": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  true,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid DeployOpts helm ={wait: true, timeout: 5, atomic: false}": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  false,
					}}
					return *s
				}(),
			},
			0,
		},
		"invalid DeployOpts helm (atomic=true but wait=false)": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    false,
						Timeout: metav1.Duration{Duration: 0},
						Atomic:  true,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid DeployOpts helm (wait=true but timeout=0)": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    true,
						Timeout: metav1.Duration{Duration: 0},
						Atomic:  false,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid DeployOpts helm (wait false but timeout defined)": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultDeployOptions = &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
						Wait:    false,
						Timeout: metav1.Duration{Duration: 5},
						Atomic:  false,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Method = "invalid"
					return *s
				}(),
			},
			1,
		},
		"valid method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Method = appskubermaticv1.HelmTemplateMethod
					return *s
				}(),
			},
			0,
		},
		"invalid missing source": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{}
					return *s
				}(),
			},
			1,
		},
		"invalid too many sources": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: validGitSource(), Helm: validHelmSource()}
					return *s
				}(),
			},
			1,
		},
		"invalid git source: remote is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "",
						Ref:         appskubermaticv1.GitReference{Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid helm source: url is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "",
						ChartName:    "chartname",
						ChartVersion: "1.2.3",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			2,
		},
		"invalid helm source: chartName is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://example.com/myrepo",
						ChartName:    "",
						ChartVersion: "1.2.3",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid helm source: chartVersion is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://example.com/myrepo",
						ChartName:    "chartname",
						ChartVersion: "",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"valid values: with comment": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultValuesBlock = `
# a yaml comment
key: value`[1:]
					return *s
				}(),
			},
			0,
		},
		"invalid values: both defaultValues and defaultValuesBlock are set": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultValues = &runtime.RawExtension{Raw: []byte(`{ "commonLabels": {"owner": "somebody"}}`)}
					s.DefaultValuesBlock = "key: value"
					return *s
				}(),
			},
			2,
		},
		"invalid values: yaml syntax error": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.DefaultValuesBlock = `invalid-yaml:
invalid:test`
					return *s
				}(),
			},
			1,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionSpec(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateHelmUrl(t *testing.T) {
	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid url with http protocol": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "http://localhost/myrepo",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid url with https protocol": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://example.com/myrepo",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid url with oci protocol": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "oci://localhost:5000/myrepo",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"invalid empty url": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			2,
		},
		"invalid url with unsupported protocol": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "ssh://localhost:5000/myrepo",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid url with only http scheme": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "http://",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			2,
		},
		"invalid url with only https scheme": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			2,
		},
		"invalid url with only oci scheme": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "oci://",
						ChartName:    "chartname",
						ChartVersion: "1.0.0",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			2,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionSpec(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateGitRef(t *testing.T) {
	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid Ref which is a branch": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid Ref which is a tag": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Tag: "v1.0"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid Ref which is a commit": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce24997c5d3d2503794", Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"valid Ref which is a commit in a branch": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Branch: "main", Commit: "bad9725e1b225d152074fce24997c5d3d2503794"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			0,
		},
		"invalid Ref is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref has tag and commit defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Tag: "v1.0", Commit: "bad9725e1b225d152074fce24997c5d3d2503794"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref has tag and branch defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Tag: "v1.0", Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is an empty branch": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Branch: ""},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is an empty tag": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Tag: ""},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is an empty commit": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: ""},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is commit which is too short": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "abc", Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is commit without branch": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce24997c5d3d2503794"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is commit which is too long": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce24997c5d3d2503794toolong", Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid Ref is commit which contains invalid char": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://example.com/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce249###5d3d2503794", Branch: "main"},
						Path:        "",
						Credentials: nil,
					}}
					return *s
				}(),
			},
			1,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionSpec(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateHelmCredentials(t *testing.T) {
	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid: no credentials": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = nil
					return *s
				}(),
			},
			0,
		},
		"valid: username and password are defined and RegistryConfigFile is undefined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Username: secretKeySelector, Password: secretKeySelector}
					return *s
				}(),
			},
			0,
		},
		"valid: username and password are undefined and RegistryConfigFile is defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{RegistryConfigFile: secretKeySelector}
					return *s
				}(),
			},
			0,
		},
		"invalid: username is defined and password is defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Username: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
		"invalid: password is defined and username is defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Password: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
		"invalid: username and password and RegistryConfigFile are defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Username: secretKeySelector, Password: secretKeySelector, RegistryConfigFile: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
		"invalid: username  and RegistryConfigFile are defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Username: secretKeySelector, RegistryConfigFile: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
		"invalid: password and RegistryConfigFile are defined": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Source.Helm.Credentials = &appskubermaticv1.HelmCredentials{Password: secretKeySelector, RegistryConfigFile: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionSpec(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateHelmSourceURL(t *testing.T) {
	testcases := []struct {
		name    string
		source  appskubermaticv1.HelmSource
		invalid bool
	}{
		// HTTP URLs

		{
			name: "HTTP URL",
			source: appskubermaticv1.HelmSource{
				URL: "http://example.com",
			},
		},
		{
			name: "HTTP URL with redundant plainHTTP=true",
			source: appskubermaticv1.HelmSource{
				URL:       "http://example.com",
				PlainHTTP: ptr.To(true),
			},
		},
		{
			name: "HTTP URL with invalid plainHTTP=false",
			source: appskubermaticv1.HelmSource{
				URL:       "http://example.com",
				PlainHTTP: ptr.To(false),
			},
			invalid: true,
		},
		{
			name: "HTTP URL with invalid insecure=true flag",
			source: appskubermaticv1.HelmSource{
				URL:      "http://example.com",
				Insecure: ptr.To(true),
			},
			invalid: true,
		},
		{
			name: "HTTP URL with invalid insecure=false flag",
			source: appskubermaticv1.HelmSource{
				URL:      "http://example.com",
				Insecure: ptr.To(false),
			},
			invalid: true,
		},
		{
			name: "HTTP URL with both invalid flags",
			source: appskubermaticv1.HelmSource{
				URL:       "http://example.com",
				Insecure:  ptr.To(false),
				PlainHTTP: ptr.To(false),
			},
			invalid: true,
		},

		// HTTPS URLs

		{
			name: "HTTPS URL",
			source: appskubermaticv1.HelmSource{
				URL: "https://example.com",
			},
		},
		{
			name: "HTTPS localhost URL",
			source: appskubermaticv1.HelmSource{
				URL: "https://localhost",
			},
			invalid: true,
		},
		{
			name: "HTTPS URL with invalid plainHTTP=true",
			source: appskubermaticv1.HelmSource{
				URL:       "https://example.com",
				PlainHTTP: ptr.To(true),
			},
			invalid: true,
		},
		{
			name: "HTTPS URL with redundant plainHTTP=false",
			source: appskubermaticv1.HelmSource{
				URL:       "https://example.com",
				PlainHTTP: ptr.To(false),
			},
		},
		{
			name: "HTTPS URL with insecure=true flag",
			source: appskubermaticv1.HelmSource{
				URL:      "https://example.com",
				Insecure: ptr.To(true),
			},
		},
		{
			name: "HTTPS URL with insecure=false flag",
			source: appskubermaticv1.HelmSource{
				URL:      "https://example.com",
				Insecure: ptr.To(false),
			},
		},

		// OCI URLs

		{
			name: "OCI URL",
			source: appskubermaticv1.HelmSource{
				URL: "oci://example.com",
			},
		},
		{
			name: "OCI localhost URL",
			source: appskubermaticv1.HelmSource{
				URL: "oci://localhost",
			},
		},
		{
			name: "OCI localhost URL with plainHTTP=false",
			source: appskubermaticv1.HelmSource{
				URL:       "oci://localhost",
				PlainHTTP: ptr.To(false),
			},
			invalid: true,
		},
		{
			name: "OCI URL with plainHTTP=true",
			source: appskubermaticv1.HelmSource{
				URL:       "oci://example.com",
				PlainHTTP: ptr.To(true),
			},
		},
		{
			name: "OCI URL with plainHTTP=false",
			source: appskubermaticv1.HelmSource{
				URL:       "oci://example.com",
				PlainHTTP: ptr.To(false),
			},
		},
		{
			name: "OCI URL with insecure=true flag",
			source: appskubermaticv1.HelmSource{
				URL:      "oci://example.com",
				Insecure: ptr.To(true),
			},
		},
		{
			name: "OCI URL with insecure=false flag",
			source: appskubermaticv1.HelmSource{
				URL:      "oci://example.com",
				Insecure: ptr.To(false),
			},
		},
		{
			name: "OCI URL with plainHTTP=true and insecure=true",
			source: appskubermaticv1.HelmSource{
				URL:       "oci://example.com",
				Insecure:  ptr.To(true),
				PlainHTTP: ptr.To(true),
			},
			invalid: true,
		},
		{
			name: "OCI URL with plainHTTP=true and insecure=false",
			source: appskubermaticv1.HelmSource{
				URL:       "oci://example.com",
				Insecure:  ptr.To(false),
				PlainHTTP: ptr.To(true),
			},
			invalid: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateHelmSourceURL(&tc.source, nil)
			if tc.invalid {
				if len(errs) == 0 {
					t.Fatal("Expected source to be invalid, but validation succeeded.")
				}
			} else if len(errs) > 0 {
				t.Fatalf("Expected source to be valid, but got errors: %v", errs.ToAggregate())
			}
		})
	}
}

func TestValidateGitCredentials(t *testing.T) {
	tt := map[string]struct {
		ad        appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"valid token method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodToken, Token: secretKeySelector}
					return *s
				}(),
			},
			0,
		},
		"valid  password method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodPassword, Username: secretKeySelector, Password: secretKeySelector}
					return *s
				}(),
			},
			0,
		},
		"valid ssh-key method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodSSHKey, SSHKey: secretKeySelector}
					return *s
				}(),
			},
			0,
		},
		"invalid token method with nil token": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodToken, Token: nil}
					return *s
				}(),
			},
			1,
		},
		"invalid password method with nil username": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodPassword, Username: nil, Password: secretKeySelector}
					return *s
				}(),
			},
			1,
		},
		"invalid password method with nil password": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodPassword, Username: secretKeySelector, Password: nil}
					return *s
				}(),
			},
			1,
		},
		"invalid password method with nil username and password": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodPassword, Username: nil, Password: nil}
					return *s
				}(),
			},
			2,
		},
		"invalid ssh-key method with nil sshKey": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodSSHKey, SSHKey: nil}
					return *s
				}(),
			},
			1,
		},
		"invalid unknown method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: "unknown"}
					return *s
				}(),
			},
			2,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.ad.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionSpec(tc.ad)

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
				{Version: "v1", Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSource()}}},
				{Version: "v1", Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSource()}}},
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

func TestValidateApplicationDefinitionUpdate(t *testing.T) {
	appDef := appskubermaticv1.ApplicationDefinition{
		Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
			s := spec.DeepCopy()
			s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{URL: "https://kubermatic.io", ChartName: "test", ChartVersion: "1.0.0"}}
			return *s
		}(),
	}

	tt := map[string]struct {
		oldAd     appskubermaticv1.ApplicationDefinition
		newAd     appskubermaticv1.ApplicationDefinition
		expErrLen int
	}{
		"Update ApplicationDefinition Success": {
			oldAd: appDef,
			newAd: appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					spec := appDef.Spec.DeepCopy()
					spec.Description = "new description"
					return *spec
				}(),
			},
			expErrLen: 0,
		},
		"Update ApplicationDefinition failure - .Spec.Method is immutable": {
			oldAd: appDef,
			newAd: appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					spec := appDef.Spec.DeepCopy()
					spec.Method = "foo"
					return *spec
				}(),
			},
			// Current we only support one Method which is helm. So we got 2 errors:
			//  1) spec.method: Unsupported value "foo"
			//  2) spec.method: Invalid value: "foo": field is immutable
			expErrLen: 2,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			tc.oldAd.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			tc.newAd.TypeMeta = metav1.TypeMeta{Kind: "ApplicationDefinition", APIVersion: "apps.kubermatic.k8c.io/v1"}
			errl := ValidateApplicationDefinitionUpdate(tc.newAd, tc.oldAd)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
			}
		})
	}
}

func TestValidateApplicationDefinitionDelete(t *testing.T) {
	notManagedAppName := "not-managed"
	managedAppName := "managed"
	appDef := getApplicationDefinition(notManagedAppName, false, false, nil, nil)
	managedAppDef := getApplicationDefinition(managedAppName, false, false, nil, map[string]string{
		appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
	})

	testCases := []struct {
		name          string
		ad            *appskubermaticv1.ApplicationDefinition
		expectedError string
	}{
		{
			name:          "scenario 1: application deletion is allowed for non-managed application definition",
			ad:            appDef,
			expectedError: "[]",
		},
		{
			name:          "scenario 2: application deletion is not allowed for managed application definition",
			ad:            managedAppDef,
			expectedError: `[metadata.labels: Invalid value: map[string]string{"apps.kubermatic.k8c.io/managed-by":"kkp"}: ` + deleteSystemAppErrorMsg() + `]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateApplicationDefinitionDelete(*testCase.ad)

			if fmt.Sprint(err) != testCase.expectedError {
				t.Fatalf("expected error to be %s but got %v", testCase.expectedError, err)
			}
		})
	}
}
