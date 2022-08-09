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
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func boolPtr(value bool) *bool {
	return &[]bool{value}[0]
}

func genPresets() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled"},
			Spec: kubermaticv1.PresetSpec{
				Enabled: boolPtr(true),
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "disabled"},
			Spec: kubermaticv1.PresetSpec{
				Enabled: boolPtr(false),
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled-do"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					ProviderPreset: kubermaticv1.ProviderPreset{
						Enabled: boolPtr(true),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "disabled-do"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					ProviderPreset: kubermaticv1.ProviderPreset{
						Enabled: boolPtr(false),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-with-dc"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					ProviderPreset: kubermaticv1.ProviderPreset{
						Datacenter: "a",
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "disabled-do-with-dc"},
			Spec: kubermaticv1.PresetSpec{
				Digitalocean: &kubermaticv1.Digitalocean{
					ProviderPreset: kubermaticv1.ProviderPreset{
						Datacenter: "a",
						Enabled:    boolPtr(false),
					},
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-with-acme-email"},
			Spec: kubermaticv1.PresetSpec{
				RequiredEmails: []string{test.RequiredEmailDomain},
				Digitalocean: &kubermaticv1.Digitalocean{
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-with-test-email"},
			Spec: kubermaticv1.PresetSpec{
				RequiredEmails: []string{"test.com"},
				Digitalocean: &kubermaticv1.Digitalocean{
					Token: "token",
				},
			},
		},
		&kubermaticv1.Preset{
			ObjectMeta: metav1.ObjectMeta{Name: "enabled-multi-provider"},
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

func sortPresets(presets []apiv2.Preset) {
	sort.Slice(presets, func(i, j int) bool {
		return strings.Compare(presets[i].Name, presets[j].Name) < 1
	})
}

func TestListPresets(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Disabled               bool
		ExpectedResponse       *apiv2.PresetList
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled presets",
			Disabled: false,
			ExpectedResponse: &apiv2.PresetList{
				Items: []apiv2.Preset{
					{Name: "enabled", Enabled: true, Providers: []apiv2.PresetProvider{}},
					{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
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
			ExpectedResponse: &apiv2.PresetList{
				Items: []apiv2.Preset{
					{Name: "enabled", Enabled: true, Providers: []apiv2.PresetProvider{}},
					{Name: "disabled", Providers: []apiv2.PresetProvider{}},
					{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
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
			// Tests need a default user otherwise, the GenDefaultAPIUser gets admin
			kubermaticObj := []ctrlruntimeclient.Object{test.GenDefaultUser()}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.HTTPStatus, res.Code)

			response := &apiv2.PresetList{}
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
		ExpectedResponse       *apiv2.PresetList
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled digitalocean presets",
			Disabled: false,
			Provider: string(kubermaticv1.DigitaloceanCloudProvider),
			ExpectedResponse: &apiv2.PresetList{
				Items: []apiv2.Preset{
					{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
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
			Provider: string(kubermaticv1.DigitaloceanCloudProvider),
			ExpectedResponse: &apiv2.PresetList{
				Items: []apiv2.Preset{
					{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do-with-dc", Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
					{Name: "disabled-do", Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
					{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
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
			Provider:   string(kubermaticv1.DigitaloceanCloudProvider),
			Datacenter: "a",
			ExpectedResponse: &apiv2.PresetList{Items: []apiv2.Preset{
				{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 4
		{
			Name:       "scenario 4: list all digitalocean presets for datacenter a",
			Disabled:   true,
			Provider:   string(kubermaticv1.DigitaloceanCloudProvider),
			Datacenter: "a",
			ExpectedResponse: &apiv2.PresetList{Items: []apiv2.Preset{
				{Name: "enabled-do", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "disabled-do", Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
				{Name: "enabled-do-with-dc", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "disabled-do-with-dc", Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider}}},
				{Name: "enabled-do-with-acme-email", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
				{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},

		// scenario 5
		{
			Name:     "scenario 5: list enabled anexia provider",
			Disabled: false,
			Provider: string(kubermaticv1.AnexiaCloudProvider),
			ExpectedResponse: &apiv2.PresetList{Items: []apiv2.Preset{
				{Name: "enabled-multi-provider", Enabled: true, Providers: []apiv2.PresetProvider{{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true}, {Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true}}},
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
			ExpectedResponse:       &apiv2.PresetList{Items: []apiv2.Preset{}},
			HTTPStatus:             http.StatusBadRequest,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresets(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/providers/%s/presets?disabled=%v&datacenter=%s", tc.Provider, tc.Disabled, tc.Datacenter), strings.NewReader(""))
			res := httptest.NewRecorder()
			// Tests need a default user otherwise, the GenDefaultAPIUser gets admin
			kubermaticObj := []ctrlruntimeclient.Object{test.GenDefaultUser()}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.HTTPStatus, res.Code)

			response := &apiv2.PresetList{}
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
		Provider        apiv2.PresetProvider
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
				ObjectMeta: metav1.ObjectMeta{Name: "disabled-preset"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(false),
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "disabled-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-preset"},
				Spec: kubermaticv1.PresetSpec{
					Enabled: boolPtr(true),
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-preset"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Enabled:    true,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "disabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(false)},
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "disabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(true)},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(true)},
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(false)},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-preset"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "enabled-do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(false)},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Enabled:    false,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "preset"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusOK {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestCreatePreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        apiv2.PresetProvider
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(false)},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.AnexiaCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "multi-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "multi-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
			Provider: apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
			Provider:        apiv2.PresetProvider{Name: "xyz", Enabled: true},
			Body:            "{}",
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 6
		{
			Name:     "scenario 6: missing provider configuration",
			Provider: apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
			Provider: apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
			Provider: apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
			Provider: apiv2.PresetProvider{Name: kubermaticv1.FakeCloudProvider, Enabled: true},
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

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusCreated {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestUpdatePreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        apiv2.PresetProvider
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(false)},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.AlibabaCloudProvider, Enabled: true},
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
				ObjectMeta: metav1.ObjectMeta{Name: "alibaba-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Alibaba: &kubermaticv1.Alibaba{
						AccessKeyID:     "test",
						AccessKeySecret: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "alibaba-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
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
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.OpenstackCloudProvider, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "openstack-preset"
					  },
					  "spec": {
						"openstack": {
						  "username": "updated",
						  "password": "updated",
						  "project": "updated",
						  "domain": "updated"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "openstack-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username:       "test",
						Password:       "test",
						Project:        "test",
						Domain:         "test",
						FloatingIPPool: "test",
						RouterID:       "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "openstack-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username: "updated",
						Password: "updated",
						Project:  "updated",
						Domain:   "updated",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 4
		{
			Name:       "scenario 4: block user from updating multiple providers at once",
			PresetName: "openstack-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.OpenstackCloudProvider, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "openstack-preset"
					  },
					  "spec": {
						"openstack": {
						  "username": "updated",
						  "password": "updated",
						  "project": "updated",
						  "domain": "updated"
						},
					  "digitalocean": {
						  "token": "updated"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "openstack-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Openstack: &kubermaticv1.Openstack{
						Username:       "test",
						Password:       "test",
						Project:        "test",
						Domain:         "test",
						FloatingIPPool: "test",
						RouterID:       "test",
					},
				},
			},
			HTTPStatus:      http.StatusBadRequest,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 5
		{
			Name:     "scenario 5: block preset update for regular user",
			Provider: apiv2.PresetProvider{Name: kubermaticv1.FakeCloudProvider, Enabled: true},
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

		// scenario 6
		{
			Name:       "scenario 6: add requiredEmails",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "do-preset"
					  },
					  "spec": {
						"requiredEmails": ["foo.bar@example.com"],
						"digitalocean": {
						  "token": "test"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					RequiredEmails: []string{"foo.bar@example.com"},
					Digitalocean: &kubermaticv1.Digitalocean{
						ProviderPreset: kubermaticv1.ProviderPreset{Enabled: boolPtr(true)},
						Token:          "test",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 7
		{
			Name:       "scenario 7: update requiredEmails",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			Body: `{
					  "metadata": {
						"name": "do-preset"
					  },
					  "spec": {
						"requiredEmails": ["foobar.com","test.com"],
						"digitalocean": {
							"token": "test"
						}
					  }
			}`,
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					// use a domain that is not the domain of the admin (i.e. acme.com)!
					RequiredEmails: []string{"foobar.com"},
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					RequiredEmails: []string{"foobar.com", "test.com"},
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},

		// scenario 8
		{
			Name:       "scenario 8: remove requiredEmails",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
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
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					// use a domain that is not the domain of the admin (i.e. acme.com)!
					RequiredEmails: []string{"foobar.com"},
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset", ResourceVersion: "1"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
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

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusCreated {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestDeleteProviderPreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        apiv2.PresetProvider
		Body            string
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		IsDeleted       bool
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:       "scenario 1: delete digitalocean preset",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
				},
			},
			ExpectedPreset:  &kubermaticv1.Preset{},
			IsDeleted:       true,
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		// scenario 2
		{
			Name:       "scenario 2: delete digitalocean preset but keep the preset",
			PresetName: "do-os-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.DigitaloceanCloudProvider, Enabled: true},
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-os-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Digitalocean: &kubermaticv1.Digitalocean{
						Token: "test",
					},
					Openstack: &kubermaticv1.Openstack{
						Username: "username",
						Password: "password",
						Project:  "project",
						Domain:   "domain",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-os-preset", ResourceVersion: "1000"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					Openstack: &kubermaticv1.Openstack{
						Username: "username",
						Password: "password",
						Project:  "project",
						Domain:   "domain",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			IsDeleted:       false,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/providers/%s/presets/%s", tc.Provider.Name, tc.PresetName), nil)
			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				if !tc.IsDeleted {
					t.Fatalf("failed to get preset: %+v", err)
				} else {
					return
				}
			}

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestDeletePresetProvider(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Provider        apiv2.PresetProvider
		Body            string
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:       "scenario 1: delete aws provider",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.AWSCloudProvider, Enabled: true},
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec: kubermaticv1.PresetSpec{
					AWS: &kubermaticv1.AWS{
						AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
						SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
						AssumeRoleARN:   "arn:aws:iam::123456789012:role/kubermaitc-test",
					},
				},
			},
			ExpectedPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		// scenario 2
		{
			Name:            "scenario 2: delete aws provider as non admin",
			PresetName:      test.TestFakeCredential,
			Provider:        apiv2.PresetProvider{Name: kubermaticv1.OpenstackCloudProvider, Enabled: true},
			ExistingPreset:  test.GenDefaultPreset(),
			HTTPStatus:      http.StatusForbidden,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:       "scenario 3: delete non-existing provider",
			PresetName: "do-preset",
			Provider:   apiv2.PresetProvider{Name: kubermaticv1.AWSCloudProvider, Enabled: true},
			ExistingPreset: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{Name: "do-preset"},
				TypeMeta:   metav1.TypeMeta{Kind: "Preset", APIVersion: "kubermatic.k8c.io/v1"},
				Spec:       kubermaticv1.PresetSpec{},
			},
			HTTPStatus:      http.StatusNotFound,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		// scenario 4
		{
			Name:            "scenario 4: delete provider from non-existing preset",
			PresetName:      "non-existing-preset",
			Provider:        apiv2.PresetProvider{Name: kubermaticv1.OpenstackCloudProvider, Enabled: true},
			ExistingPreset:  test.GenDefaultPreset(),
			HTTPStatus:      http.StatusNotFound,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/presets/%s/provider/%s", tc.PresetName, tc.Provider.Name), nil)

			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusOK {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				t.Fatalf("failed to get preset: %+v", err)
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestDeletePreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		PresetName      string
		Body            string
		ExistingPreset  *kubermaticv1.Preset
		ExpectedPreset  *kubermaticv1.Preset
		IsDeleted       bool
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:            "scenario 1: delete preset",
			PresetName:      test.TestFakeCredential,
			ExistingPreset:  test.GenDefaultPreset(),
			IsDeleted:       true,
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
		// scenario 2
		{
			Name:            "scenario 2: delete preset as non-admin",
			PresetName:      test.TestFakeCredential,
			ExistingPreset:  test.GenDefaultPreset(),
			HTTPStatus:      http.StatusForbidden,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:            "scenario 3: delete non-existing preset",
			PresetName:      "non-existing-preset",
			ExistingPreset:  test.GenDefaultPreset(),
			HTTPStatus:      http.StatusNotFound,
			ExistingAPIUser: test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2/presets/%s", tc.PresetName), nil)
			res := httptest.NewRecorder()

			existingKubermaticObjs := []ctrlruntimeclient.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}
			if tc.ExistingPreset != nil {
				existingKubermaticObjs = append(existingKubermaticObjs, tc.ExistingPreset)
			}

			ep, clientSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []ctrlruntimeclient.Object{}, []ctrlruntimeclient.Object{}, existingKubermaticObjs, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)
			assert.Equal(t, tc.HTTPStatus, res.Code)

			if res.Code != http.StatusOK {
				return
			}

			preset := &kubermaticv1.Preset{}
			if err := clientSets.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Namespace: "", Name: tc.PresetName}, preset); err != nil {
				if !tc.IsDeleted {
					t.Fatalf("failed to get preset: %+v", err)
				} else {
					return
				}
			}

			tc.ExpectedPreset.ResourceVersion = preset.ResourceVersion

			if !diff.SemanticallyEqual(tc.ExpectedPreset, preset) {
				t.Fatalf("Got different preset than expected:\n%v", diff.ObjectDiff(tc.ExpectedPreset, preset))
			}
		})
	}
}

func TestPresetStats(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		PresetName             string
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExpectedResponse       string
	}{
		{
			Name:             "scenario 1: test preset stats when cluster and template created with preset",
			ExpectedResponse: `{"associatedClusters":1,"associatedClusterTemplates":1}`,
			PresetName:       test.GenDefaultPreset().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
				test.GenDefaultPreset(),
				func() *kubermaticv1.Cluster {
					c := test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					c.Labels[kubermaticv1.IsCredentialPresetLabelKey] = "true"
					c.Annotations = map[string]string{kubermaticv1.PresetNameAnnotation: test.GenDefaultPreset().Name}
					return c
				}(),
				func() *kubermaticv1.ClusterTemplate {
					t := test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email)
					t.Labels[kubermaticv1.IsCredentialPresetLabelKey] = "true"
					t.Annotations = map[string]string{kubermaticv1.PresetNameAnnotation: test.GenDefaultPreset().Name}
					return t
				}(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: cluster and template created with credentials",
			ExpectedResponse: `{"associatedClusters":0,"associatedClusterTemplates":0}`,
			PresetName:       test.GenDefaultPreset().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
				test.GenDefaultPreset(),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				test.GenClusterTemplate("ct1", "ctID1", test.GenDefaultProject().Name, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/presets/%s/stats", tc.PresetName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}
