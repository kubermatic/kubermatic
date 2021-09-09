/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package version_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetClusterUpgrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		apiUser                apiv1.User
		provider               kubermaticv1.ProviderType
		versions               []*semver.Version
		updates                []operatorv1alpha1.Update
		incompatibilities      []operatorv1alpha1.Incompatibility
		wantVersions           []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available and compatible with the given provider",
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			apiUser:  *test.GenDefaultAPIUser(),
			provider: kubermaticv1.ProviderAWS,
			wantVersions: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.21.0"),
				},
				{
					Version: semver.MustParse("1.21.1"),
				},
				{
					Version: semver.MustParse("1.22.0"),
				},
				{
					Version: semver.MustParse("1.22.1"),
				},
			},
			versions: []*semver.Version{
				semver.MustParse("1.21.0"),
				semver.MustParse("1.21.1"),
				semver.MustParse("1.22.0"),
				semver.MustParse("1.22.1"),
			},
			updates: []operatorv1alpha1.Update{
				{
					From: "1.21.*",
					To:   "1.21.*",
				},
				{
					From: "1.21.*",
					To:   "1.22.*",
				},
				{
					From: "1.22.*",
					To:   "1.22.*",
				},
			},
			incompatibilities: []operatorv1alpha1.Incompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Condition: operatorv1alpha1.AlwaysCondition,
					Operation: operatorv1alpha1.CreateOperation,
				},
			},
		},
		{
			name: "upgrade available but incompatible with the given provider",
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			apiUser:  *test.GenDefaultAPIUser(),
			provider: kubermaticv1.ProviderVSphere,
			wantVersions: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.21.0"),
				},
				{
					Version: semver.MustParse("1.21.1"),
				},
			},
			versions: []*semver.Version{
				semver.MustParse("1.21.0"),
				semver.MustParse("1.21.1"),
				semver.MustParse("1.22.0"),
				semver.MustParse("1.22.1"),
			},
			updates: []operatorv1alpha1.Update{
				{
					From: "1.21.*",
					To:   "1.21.*",
				},
				{
					From: "1.21.*",
					To:   "1.22.*",
				},
				{
					From: "1.22.*",
					To:   "1.22.*",
				},
			},
			incompatibilities: []operatorv1alpha1.Incompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Condition: operatorv1alpha1.AlwaysCondition,
					Operation: operatorv1alpha1.CreateOperation,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dummyKubermaticConfiguration := operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: test.KubermaticNamespace,
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Versions: operatorv1alpha1.KubermaticVersionsConfiguration{
						Kubernetes: operatorv1alpha1.KubermaticVersioningConfiguration{
							Versions:                  tc.versions,
							Updates:                   tc.updates,
							ProviderIncompatibilities: tc.incompatibilities,
						},
					},
				},
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/providers/%s/versions", tc.provider), nil)
			res := httptest.NewRecorder()
			var machineObj []ctrlruntimeclient.Object

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.apiUser, nil, []ctrlruntimeclient.Object{}, machineObj, tc.existingKubermaticObjs, &dummyKubermaticConfiguration, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create tc endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var gotVersions []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotVersions)
			if err != nil {
				t.Fatal(err)
			}

			test.CompareVersions(t, gotVersions, tc.wantVersions)
		})
	}
}
