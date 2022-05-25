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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cs                = appskubermaticv1.ApplicationConstraints{K8sVersion: ">1.0.0", KKPVersion: ">1.0.0"}
	secretKeySelector = &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "git-cred"}, Key: "thekey"}
	helmv             = appskubermaticv1.ApplicationVersion{Version: "v1", Constraints: cs, Template: appskubermaticv1.ApplicationTemplate{Method: appskubermaticv1.HelmTemplateMethod, Source: appskubermaticv1.ApplicationSource{Helm: validHelmSouce()}}}
	gitv              = appskubermaticv1.ApplicationVersion{Version: "v2", Constraints: cs, Template: appskubermaticv1.ApplicationTemplate{Method: appskubermaticv1.HelmTemplateMethod, Source: appskubermaticv1.ApplicationSource{Git: validGitSource()}}}
	spec              = appskubermaticv1.ApplicationDefinitionSpec{Versions: []appskubermaticv1.ApplicationVersion{helmv, gitv}}
)

func validHelmSouce() *appskubermaticv1.HelmSource {
	return &appskubermaticv1.HelmSource{
		URL:          "http://localhost/charts",
		ChartName:    "apache",
		ChartVersion: "9.1.3",
		Credentials:  nil,
	}
}

func validGitSource() *appskubermaticv1.GitSource {
	return &appskubermaticv1.GitSource{
		Remote: "https://localhost/repo.git",
		Ref: appskubermaticv1.GitReference{
			Branch: "master",
		},
		Path:        "/",
		Credentials: nil,
	}
}

func TestValidateApplicationDefinition(t *testing.T) {
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					return *s
				}(),
			},
			0,
		},
		"invalid missing source": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: validGitSource(), Helm: validHelmSouce()}
					return *s
				}(),
			},
			1,
		},
		"invalid git source: remote is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "",
						Ref:         appskubermaticv1.GitReference{Branch: "master"},
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "",
						ChartName:    "chartname",
						ChartVersion: "1.2.3",
						Credentials:  nil,
					}}
					return *s
				}(),
			},
			1,
		},
		"invalid helm source: chartName is empty": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://localhost/myrepo",
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Helm: &appskubermaticv1.HelmSource{
						URL:          "https://localhost/myrepo",
						ChartName:    "chartname",
						ChartVersion: "",
						Credentials:  nil,
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
			errl := ValidateApplicationDefinition(tc.ad)

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
		"valid Ref whcih is a branch": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Branch: "master"},
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce24997c5d3d2503794"},
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Branch: "master", Commit: "bad9725e1b225d152074fce24997c5d3d2503794"},
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = "helm"
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = "helm"
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Tag: "v1.0", Branch: "master"},
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
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
					s.Versions[0].Template.Method = "helm"
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "abc"},
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
					s.Versions[0].Template.Method = "helm"
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce24997c5d3d2503794toolong"},
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
					s.Versions[0].Template.Method = "helm"
					s.Versions[0].Template.Source = appskubermaticv1.ApplicationSource{Git: &appskubermaticv1.GitSource{
						Remote:      "https://localhost/repo.git",
						Ref:         appskubermaticv1.GitReference{Commit: "bad9725e1b225d152074fce249###5d3d2503794"},
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
			errl := ValidateApplicationDefinition(tc.ad)

			if len(errl) != tc.expErrLen {
				t.Errorf("expected errLen %d, got %d. Errors are %q", tc.expErrLen, len(errl), errl)
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
					s.Versions[1].Template.Source.Git.Credentials = &appskubermaticv1.GitCredentials{Method: appskubermaticv1.GitAuthMethodSSHKey, SSHKey: nil}
					return *s
				}(),
			},
			1,
		},
		"invalid unknow method": {
			appskubermaticv1.ApplicationDefinition{
				Spec: func() appskubermaticv1.ApplicationDefinitionSpec {
					s := spec.DeepCopy()
					s.Versions[1].Template.Method = appskubermaticv1.HelmTemplateMethod
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
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "1"}, Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSouce()}, Method: appskubermaticv1.HelmTemplateMethod}},
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "1"}, Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSouce()}, Method: appskubermaticv1.HelmTemplateMethod}},
			},
			1,
		},
		"invalid kkp version": {
			[]appskubermaticv1.ApplicationVersion{
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "1", KKPVersion: "not-semver"}, Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSouce()}, Method: appskubermaticv1.HelmTemplateMethod}},
			},
			1,
		},
		"invalid k8s version": {
			[]appskubermaticv1.ApplicationVersion{
				{Version: "v1", Constraints: appskubermaticv1.ApplicationConstraints{K8sVersion: "not-semver", KKPVersion: "1"}, Template: appskubermaticv1.ApplicationTemplate{Source: appskubermaticv1.ApplicationSource{Helm: validHelmSouce()}, Method: appskubermaticv1.HelmTemplateMethod}},
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
