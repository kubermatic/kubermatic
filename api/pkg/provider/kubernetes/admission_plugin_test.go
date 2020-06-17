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

package kubernetes_test

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/semver"

	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListAdmissionPluginsFromVersion(t *testing.T) {
	t.Parallel()

	version113, err := semver.NewSemver("v1.13")
	if err != nil {
		t.Fatal(err)
	}
	version116, err := semver.NewSemver("v1.16")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		name           string
		fromVersion    string
		plugins        []runtime.Object
		expectedError  string
		expectedResult []string
	}{
		{
			name:        "test 1: get plugins for version 1.12",
			fromVersion: "1.12",
			plugins: []runtime.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "imagePolicyWebhook",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "ImagePolicyWebhook",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "runtimeClass",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "RuntimeClass",
						FromVersion: version116,
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				},
			},
			expectedResult: []string{"DefaultTolerationSeconds", "ImagePolicyWebhook"},
		},
		{
			name:        "test 1: get plugins for version 1.14.3",
			fromVersion: "1.14.3",
			plugins: []runtime.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "imagePolicyWebhook",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "ImagePolicyWebhook",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "runtimeClass",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "RuntimeClass",
						FromVersion: version116,
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				},
			},
			expectedResult: []string{"DefaultTolerationSeconds", "ImagePolicyWebhook", "EventRateLimit"},
		},
		{
			name:        "test 1: get plugins for version 1.16.0",
			fromVersion: "1.16.0",
			plugins: []runtime.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "imagePolicyWebhook",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "ImagePolicyWebhook",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "runtimeClass",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "RuntimeClass",
						FromVersion: version116,
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				},
			},
			expectedResult: []string{"DefaultTolerationSeconds", "ImagePolicyWebhook", "RuntimeClass", "EventRateLimit"},
		},
		{
			name:        "test 1: get plugins for version 1.17.0",
			fromVersion: "1.17.0",
			plugins: []runtime.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "defaultTolerationSeconds",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "DefaultTolerationSeconds",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "imagePolicyWebhook",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "ImagePolicyWebhook",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "runtimeClass",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "RuntimeClass",
						FromVersion: version116,
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: v1.ObjectMeta{
						Name: "eventRateLimit",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "EventRateLimit",
						FromVersion: version113,
					},
				},
			},
			expectedResult: []string{"DefaultTolerationSeconds", "ImagePolicyWebhook", "RuntimeClass", "EventRateLimit"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.plugins...)
			provider := kubernetes.NewAdmissionPluginsProvider(context.Background(), fakeClient)

			result, err := provider.ListPluginNamesFromVersion(tc.fromVersion)

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected: %s, got %v", tc.expectedError, err)
				}

			} else if !equality.Semantic.DeepEqual(result, tc.expectedResult) {
				t.Fatalf("expected: %v, got %v", tc.expectedResult, result)
			}
		})
	}
}
