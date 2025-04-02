/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	"reflect"
	"testing"
	"time"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/helmclient"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetReleaseName(t *testing.T) {
	tests := []struct {
		testName            string
		appNamespace        string
		appName             string
		expectedReleaseName string
	}{
		{
			testName:            "when len (namespaceName) <= 53 then namespaceName is returned",
			appNamespace:        "default",
			appName:             "app1",
			expectedReleaseName: "default-app1",
		},
		{
			testName:            "when len(namespaceName) > 53 and len(appName) <=43  then appName-sha1(namespace)[:9] is returned",
			appNamespace:        "default-012345678901234567890123456789001234567890123456789",
			appName:             "app1",
			expectedReleaseName: "app1-8232574ba",
		},

		{
			testName:            "when len(namespaceName) > 53 and len(appName) >43  then appName[:43]-sha1(namespace)[:9] is returned",
			appNamespace:        "default",
			appName:             "application-installation-super-long-name-that-should-be-truncated",
			expectedReleaseName: "application-installation-super-long-name-th-7505d64a5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			appInstallation := &appskubermaticv1.ApplicationInstallation{ObjectMeta: metav1.ObjectMeta{
				Namespace: tt.appNamespace,
				Name:      tt.appName,
			}}

			res := getReleaseName(appInstallation)
			if res != tt.expectedReleaseName {
				t.Errorf("getReleaseName() = %v, want %v", res, tt.expectedReleaseName)
			}

			size := len(res)
			if size > 53 {
				t.Errorf("getReleaseName() size should be less or equals to 53. got %v", size)
			}
		})
	}
}

func TestGetDeployOps(t *testing.T) {
	tests := []struct {
		name          string
		appDefinition *appskubermaticv1.ApplicationDefinition
		appInstall    *appskubermaticv1.ApplicationInstallation
		want          *helmclient.DeployOpts
		wantErr       bool
	}{
		{
			name: "case 1: deployOpts defined at applicationInstallation have priority (appDef.spec.DefaultDeployOptions.helm is defined)",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      false,
					Timeout:   metav1.Duration{Duration: 0},
					Atomic:    false,
					EnableDNS: false,
				}}},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      true,
					Timeout:   metav1.Duration{Duration: 1000},
					Atomic:    true,
					EnableDNS: true,
				}}},
			},
			want:    newDeployOpts(t, true, 1000, true, true),
			wantErr: false,
		},
		{
			name: "case 2: deployOpts defined at applicationInstallation have priority (appDef.spec.DefaultDeployOptions.helm is nil)",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: nil}},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      true,
					Timeout:   metav1.Duration{Duration: 1000},
					Atomic:    true,
					EnableDNS: true,
				}}},
			},
			want:    newDeployOpts(t, true, 1000, true, true),
			wantErr: false,
		},
		{
			name: "case 3: deployOpts defined at applicationInstallation have priority (appDef.spec.DefaultDeployOptions is nil)",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: nil},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      true,
					Timeout:   metav1.Duration{Duration: 1000},
					Atomic:    true,
					EnableDNS: true,
				}}},
			},
			want:    newDeployOpts(t, true, 1000, true, true),
			wantErr: false,
		},
		{
			name: "case 4: fallback to deployOpts defined in applicationDefinition when deployOpts is not defined in applicationInstallation (appInstall.spec.DefaultDeployOptions.helm is nil)",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      true,
					Timeout:   metav1.Duration{Duration: 500},
					Atomic:    false,
					EnableDNS: true,
				}}},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: &appskubermaticv1.DeployOptions{Helm: nil}},
			},
			want:    newDeployOpts(t, true, 500, false, true),
			wantErr: false,
		},
		{
			name: "case 5: fallback to deployOpts defined in applicationDefinition when deployOpts is not defined in applicationInstallation(appInstall.spec.DefaultDeployOptions is nil)",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:      true,
					Timeout:   metav1.Duration{Duration: 500},
					Atomic:    false,
					EnableDNS: true,
				}}},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: nil},
			},
			want:    newDeployOpts(t, true, 500, false, true),
			wantErr: false,
		},
		{
			name: "case 6: fallback to default when deployOpts is not defined in applicationInstallation and applicationDefinition",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: nil},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: nil},
			},
			want:    newDeployOpts(t, false, 0, false, false),
			wantErr: false,
		},
		{
			name: "case 7: error should be return if DeployOps defined at applicationInstallation level are invalid",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: nil},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:    true,
					Timeout: metav1.Duration{Duration: 0}, // if wait = true then timeout must be > 0
					Atomic:  true,
				}}},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "case 8: error should be return if DeployOps defined at applicationDefinition level are invalid",
			appDefinition: &appskubermaticv1.ApplicationDefinition{
				Spec: appskubermaticv1.ApplicationDefinitionSpec{DefaultDeployOptions: &appskubermaticv1.DeployOptions{Helm: &appskubermaticv1.HelmDeployOptions{
					Wait:    true,
					Timeout: metav1.Duration{Duration: 0}, // if wait = true then timeout must be > 0
					Atomic:  true,
				}}},
			},
			appInstall: &appskubermaticv1.ApplicationInstallation{
				Spec: appskubermaticv1.ApplicationInstallationSpec{DeployOptions: nil},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDeployOpts(tt.appDefinition, tt.appInstall)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getDeployOpts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("getDeployOpts() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func newDeployOpts(t *testing.T, wait bool, timeout time.Duration, atomic bool, enableDNS bool) *helmclient.DeployOpts {
	t.Helper()
	deployOps, err := helmclient.NewDeployOpts(wait, timeout, atomic, enableDNS)
	if err != nil {
		t.Fatalf("failed to build deployOpts: %s", err)
	}
	return deployOps
}
