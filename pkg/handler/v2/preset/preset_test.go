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

package preset_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	v2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func boolPtr(value bool) *bool {
	return &[]bool{value}[0]
}

func genPresets() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled"},
			Spec: kubermaticv1.PresetSpec{
				Enabled: boolPtr(true),
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "disabled"},
			Spec: kubermaticv1.PresetSpec{
				Enabled: boolPtr(false),
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled-do"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					PresetProvider: kubermaticv1.PresetProvider{
						Enabled: boolPtr(true),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "disabled-do"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					PresetProvider: kubermaticv1.PresetProvider{
						Enabled: boolPtr(false),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled-do-with-dc"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					PresetProvider: kubermaticv1.PresetProvider{
						Datacenter: "a",
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "disabled-do-with-dc"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					PresetProvider: kubermaticv1.PresetProvider{
						Datacenter: "a",
						Enabled:    boolPtr(false),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled-do-with-acme-email"},
			Spec: kubermaticv1.PresetSpec{
				RequiredEmailDomain: test.RequiredEmailDomain,
				Digitalocean: &kubermaticv1.Digitalocean{
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled-do-with-test-email"},
			Spec: kubermaticv1.PresetSpec{
				RequiredEmailDomain: "test.com",
				Digitalocean: &kubermaticv1.Digitalocean{
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: v1.ObjectMeta{Name: "enabled-multi-provider"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					Token: "token",
				},
				Anexia: &kubermaticv1.Anexia{
					Token: "token",
				},
			},
		},
	}
}

func sortPresets(presets []v2.Preset) {
	sort.Slice(presets, func(i, j int) bool {
		return strings.Compare(presets[i].Name, presets[j].Name) < 1
	})
}

func TestListPresets(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Disabled               bool
		ExpectedResponse       *v2.PresetList
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled presets",
			Disabled: false,
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{Name: "enabled", Enabled: true, Providers: []v2.PresetProvider{}},
					{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 2
		{
			Name:     "scenario 2: list all presets",
			Disabled: true,
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{Name: "enabled", Enabled: true, Providers: []v2.PresetProvider{}},
					{Name: "disabled", Providers: []v2.PresetProvider{}},
					{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/presets?disabled=%v", tc.Disabled), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := make([]ctrlruntimeclient.Object, 0)
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.HTTPStatus, res.Code)

			response := &v2.PresetList{}
			err = json.Unmarshal(res.Body.Bytes(), response)
			if err != nil {
				t.Fatal(err)
			}

			// API response is sorted by default so apply sort to our expected response also to avoid manual sort
			sortPresets(tc.ExpectedResponse.Items)
			if !reflect.DeepEqual(tc.ExpectedResponse, response) {
				t.Errorf("expected\n%+v\ngot\n%+v", tc.ExpectedResponse, response)
			}
		})
	}
}

func TestListProviderPresets(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Disabled               bool
		Provider               string
		Datacenter             string
		ExpectedResponse       *v2.PresetList
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled digitalocean presets",
			Disabled: false,
			Provider: string(kubermaticv1.ProviderDigitalocean),
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 2
		{
			Name:     "scenario 2: list all digitalocean presets",
			Disabled: true,
			Provider: string(kubermaticv1.ProviderDigitalocean),
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do-with-dc", Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
					{Name: "disabled-do", Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 3
		{
			Name:       "scenario 3: list enabled digitalocean presets for datacenter a",
			Disabled:   false,
			Provider:   string(kubermaticv1.ProviderDigitalocean),
			Datacenter: "a",
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 4
		{
			Name:       "scenario 4: list all digitalocean presets for datacenter a",
			Disabled:   true,
			Provider:   string(kubermaticv1.ProviderDigitalocean),
			Datacenter: "a",
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{Name: "enabled-do", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "disabled-do", Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
				{Name: "enabled-do-with-dc", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "disabled-do-with-dc", Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean}}},
				{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}}},
				{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 5
		{
			Name:     "scenario 5: list enabled anexia provider",
			Disabled: false,
			Provider: string(kubermaticv1.ProviderAnexia),
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{Name: "enabled-multi-provider", Enabled: true, Providers: []v2.PresetProvider{{Name: kubermaticv1.ProviderDigitalocean, Enabled: true}, {Name: kubermaticv1.ProviderAnexia, Enabled: true}}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 6
		{
			Name:                   "scenario 6: invalid provider name",
			Disabled:               true,
			Provider:               "invalid",
			ExpectedResponse:       &v2.PresetList{Items: []v2.Preset{}},
			HTTPStatus:             http.StatusBadRequest,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/providers/%s/presets?disabled=%v&datacenter=%s", tc.Provider, tc.Disabled, tc.Datacenter), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := make([]ctrlruntimeclient.Object, 0)
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.HTTPStatus, res.Code)

			response := &v2.PresetList{}
			err = json.Unmarshal(res.Body.Bytes(), response)
			if err != nil {
				t.Fatal(err)
			}

			// API response is sorted by default so apply sort to our expected response also to avoid manual sort
			sortPresets(tc.ExpectedResponse.Items)
			if res.Code == http.StatusOK && !reflect.DeepEqual(tc.ExpectedResponse, response) {
				t.Errorf("expected\n%+v\ngot\n%+v", tc.ExpectedResponse, response)
			}
		})
	}
}

func TestUpdatePresetStatus(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Enabled         bool
		Provider        v2.PresetProvider
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:       "scenario 1: enable disabled preset",
			PresetName: "disabled-preset",
			Enabled:    true,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "disabled-preset"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(false),
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "disabled-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(true),
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 2
		{
			Name:       "scenario 2: disable enabled preset",
			PresetName: "enabled-preset",
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-preset"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(true),
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(false),
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 3
		{
			Name:       "scenario 3: disable enabled preset with no enabled status set",
			PresetName: "enabled-preset",
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-preset"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(false),
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 4
		{
			Name:       "scenario 4: enable disabled digitalocean preset",
			PresetName: "disabled-do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Enabled:    true,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "disabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(false)},
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "disabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(true)},
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 5
		{
			Name:       "scenario 5: disable enabled digitalocean preset",
			PresetName: "enabled-do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(true)},
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(false)},
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 5
		{
			Name:       "scenario 6: disable enabled digitalocean preset with no enabled status set",
			PresetName: "enabled-do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "enabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(false)},
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 6
		{
			Name:            "scenario 6: block preset update for regular user",
			PresetName:      "enabled-preset",
			Enabled:         false,
			HTTPStatus:      http.StatusForbidden,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},

		// scenario 7
		{
			Name:       "scenario 6: block status update when provider configuration missing",
			PresetName: "preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "preset"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			HTTPStatus:      http.StatusConflict,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v2/presets/%s/status?provider=%s", tc.PresetName, tc.Provider.Name), strings.NewReader(fmt.Sprintf(`{"enabled": %v}`, tc.Enabled)))
			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusOK {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.TODO(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if diff := deep.Equal(tc.ExpectedPreset, preset); diff != nil {
				t.Errorf("Got different preset than expected.\nDiff: %v", diff)
			}
		})
	}
}

func TestCreatePreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        v2.PresetProvider
		Body            string
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:       "scenario 1: create digitalocean preset",
			PresetName: "do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "do-preset"
					  },
					  "spec": {
						"digitalocean": {
						  "token": "test"
						}
					  }
			}`,
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{Token: "test"},
				},
			},
			HTTPStatus:      http.StatusCreated,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 2
		{
			Name:       "scenario 2: create disabled digitalocean preset",
			PresetName: "do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "do-preset"
					  },
					  "spec": {
						"digitalocean": {
						  "token": "test",
						  "enabled": false
						}
					  }
			}`,
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(false)},
						Token:          "test",
					},
				},
			},
			HTTPStatus:      http.StatusCreated,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 3
		{
			Name:       "scenario 3: add new anexia provider to existing preset",
			PresetName: "multi-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderAnexia, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "multi-preset"
					  },
					  "spec": {
						"anexia": {
						  "token": "test"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "multi-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "multi-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Anexia: &kubermaticv1.Anexia{
						Token: "test",
					},
				},
			},
			HTTPStatus:      http.StatusCreated,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 4
		{
			Name:     "scenario 4: block overriding existing preset provider configuration",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "do-preset"
					 },
					 "spec": {
						"digitalocean": {
						  "token": "updated"
						}
					 }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			HTTPStatus:      http.StatusConflict,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 5
		{
			Name:            "scenario 5: provided invalid provider name",
			Provider:        v2.PresetProvider{Name: "xyz", Enabled: true},
			Body:            "{}",
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 6
		{
			Name:     "scenario 6: missing provider configuration",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "do-preset"
					 },
					 "spec": {}
			}`,
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 7
		{
			Name:     "scenario 7: missing required token field for digitalocean provider",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "do-preset"
					 },
					 "spec": { "digitalocean": {} }
			}`,
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 8
		{
			Name:     "scenario 8: unexpected provider configuration when creating digitalocean preset",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "do-preset"
					 },
					 "spec": {
						"digitalocean": { "token": "test" },
						"anexia": { "token": "test" }
					 }
			}`,
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 9
		{
			Name:     "scenario 9: block preset creation for regular user",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderFake, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "fake-preset"
					 },
					 "spec": {
						"fake": {
						  "token": "test"
						}
					 }
			}`,
			HTTPStatus:      http.StatusForbidden,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v2/providers/%s/presets", tc.Provider.Name), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusCreated {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.TODO(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if diff := deep.Equal(tc.ExpectedPreset, preset); diff != nil {
				t.Errorf("Got different preset than expected.\nDiff: %v", diff)
			}
		})
	}
}

func TestUpdatePreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        v2.PresetProvider
		Body            string
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:       "scenario 1: update digitalocean preset token and disable it",
			PresetName: "do-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderDigitalocean, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "do-preset"
					  },
					  "spec": {
						"digitalocean": {
						  "token": "updated",
						  "enabled": false
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						PresetProvider: kubermaticv1.PresetProvider{Enabled: boolPtr(false)},
						Token:          "updated",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 2
		{
			Name:       "scenario 2: update alibaba credentials",
			PresetName: "alibaba-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderAlibaba, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "alibaba-preset"
					  },
					  "spec": {
						"alibaba": {
						  "accessKeyId": "updated",
						  "accessKeySecret": "updated"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "alibaba-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Alibaba: &kubermaticv1.Alibaba{
						AccessKeyID:     "test",
						AccessKeySecret: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "alibaba-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Alibaba: &kubermaticv1.Alibaba{
						AccessKeyID:     "updated",
						AccessKeySecret: "updated",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 3
		{
			Name:       "scenario 3: omit optional openstack fields to remove them",
			PresetName: "openstack-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderOpenstack, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "openstack-preset"
					  },
					  "spec": {
						"openstack": {
						  "username": "updated",
						  "password": "updated",
						  "tenant": "updated",
						  "domain": "updated"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "openstack-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username:       "test",
						Password:       "test",
						TenantID:       "test",
						Domain:         "test",
						FloatingIPPool: "test",
						RouterID:       "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "openstack-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username: "updated",
						Password: "updated",
						TenantID: "updated",
						Domain:   "updated",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 4
		{
			Name:       "scenario 4: replace username/password with application credentials",
			PresetName: "openstack-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderOpenstack, Enabled: true},
			Body: `{
						"metadata": {
						"name": "openstack-preset"
						},
						"spec": {
						"openstack": {
							"applicationCredentialID": "updated",
							"applicationCredentialSecret": "updated",
							"domain": "updated"
						}
						}
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "openstack-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username:       "test",
						Password:       "test",
						TenantID:       "test",
						Domain:         "test",
						FloatingIPPool: "test",
						RouterID:       "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "openstack-preset", ResourceVersion: "1"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						ApplicationCredentialID:     "updated",
						ApplicationCredentialSecret: "updated",
						Domain:                      "updated",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 5
		{
			Name:       "scenario 5: block user from updating multiple providers at once",
			PresetName: "openstack-preset",
			Provider:   v2.PresetProvider{Name: kubermaticv1.ProviderOpenstack, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "openstack-preset"
					  },
					  "spec": {
						"openstack": {
						  "username": "updated",
						  "password": "updated",
						  "tenant": "updated",
						  "domain": "updated"
						},
					  "digitalocean": {
						  "token": "updated"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: v1.ObjectMeta{Name: "openstack-preset"},
				TypeMeta:   v1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8s.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Openstack: &kubermaticv1.Openstack{
						Username:       "test",
						Password:       "test",
						TenantID:       "test",
						Domain:         "test",
						FloatingIPPool: "test",
						RouterID:       "test",
					},
				},
			},
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 6
		{
			Name:     "scenario 6: block preset update for regular user",
			Provider: v2.PresetProvider{Name: kubermaticv1.ProviderFake, Enabled: true},
			Body: `{
					 "metadata": {
						"name": "fake-preset"
					 },
					 "spec": {
						"fake": {
						  "token": "test"
						}
					 }
			}`,
			HTTPStatus:      http.StatusForbidden,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v2/providers/%s/presets", tc.Provider.Name), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusCreated {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.TODO(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			if diff := deep.Equal(tc.ExpectedPreset, preset); diff != nil {
				t.Errorf("Got different preset than expected.\nDiff: %v", diff)
			}
		})
	}
}
