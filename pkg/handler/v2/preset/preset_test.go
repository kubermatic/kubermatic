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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	v2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
)

func boolPtr(value bool) *bool {
	return &[]bool{value}[0]
}

func genPresetList() []runtime.Object {
	return []runtime.Object{
		&kubermaticv1.PresetList{
			Items: []kubermaticv1.Preset{
				{
					ObjectMeta: v1.ObjectMeta{Name: "enabled"},
					Spec: kubermaticv1.PresetSpec{
						Enabled: boolPtr(true),
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{Name: "disabled"},
					Spec: kubermaticv1.PresetSpec{
						Enabled: boolPtr(false),
					},
				},
				{
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
				{
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
				{
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
				{
					ObjectMeta: v1.ObjectMeta{Name: "disabled-do-with-dc"},
					Spec: kubermaticv1.PresetSpec{
						Digitalocean: &kubermaticv1.Digitalocean{
							PresetProvider: kubermaticv1.PresetProvider{
								Datacenter: "a",
								Enabled: boolPtr(false),
							},
							Token: "token",
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{Name: "enabled-do-with-acme-email"},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: test.RequiredEmailDomain,
						Digitalocean: &kubermaticv1.Digitalocean{
							Token: "token",
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{Name: "enabled-do-with-test-email"},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Digitalocean: &kubermaticv1.Digitalocean{
							Token: "token",
						},
					},
				},
				{
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
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled presets",
			Disabled: false,
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{"enabled", true, []kubermaticv1.ProviderType{}},
					{"enabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-acme-email", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-multi-provider", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean, kubermaticv1.ProviderAnexia}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 2
		{
			Name:     "scenario 2: list all presets",
			Disabled: true,
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{"enabled", true, []kubermaticv1.ProviderType{}},
					{"disabled", false, []kubermaticv1.ProviderType{}},
					{"enabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-acme-email", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-multi-provider", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean, kubermaticv1.ProviderAnexia}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/presets?disabled=%v", tc.Disabled), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := make([]runtime.Object, 0)
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:     "scenario 1: list enabled digitalocean presets",
			Disabled: false,
			Provider: string(kubermaticv1.ProviderDigitalocean),
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{"enabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-acme-email", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-multi-provider", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean, kubermaticv1.ProviderAnexia}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 2
		{
			Name:     "scenario 2: list all digitalocean presets",
			Disabled: true,
			Provider: string(kubermaticv1.ProviderDigitalocean),
			ExpectedResponse: &v2.PresetList{
				Items: []v2.Preset{
					{"enabled-do", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do-with-dc", false, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-do-with-acme-email", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"disabled-do", false, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
					{"enabled-multi-provider", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean, kubermaticv1.ProviderAnexia}},
				},
			},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 3
		{
			Name:       "scenario 3: list enabled digitalocean presets for datacenter a",
			Disabled:   false,
			Provider:   string(kubermaticv1.ProviderDigitalocean),
			Datacenter: "a",
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 4
		{
			Name:       "scenario 4: list all digitalocean presets for datacenter a",
			Disabled:   true,
			Provider:   string(kubermaticv1.ProviderDigitalocean),
			Datacenter: "a",
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{"enabled-do-with-dc", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
				{"disabled-do-with-dc", false, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 5
		{
			Name:     "scenario 5: list enabled anexia provider",
			Disabled: false,
			Provider: string(kubermaticv1.ProviderAnexia),
			ExpectedResponse: &v2.PresetList{Items: []v2.Preset{
				{"enabled-multi-provider", true, []kubermaticv1.ProviderType{kubermaticv1.ProviderDigitalocean, kubermaticv1.ProviderAnexia}},
			}},
			HTTPStatus:             http.StatusOK,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},

		// scenario 6
		{
			Name:                   "scenario 6: invalid provider name",
			Disabled:               true,
			Provider:               "invalid",
			ExpectedResponse:       &v2.PresetList{Items: []v2.Preset{}},
			HTTPStatus:             http.StatusBadRequest,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: genPresetList(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/providers/%s/presets?disabled=%v&datacenter=%s", tc.Provider, tc.Disabled, tc.Datacenter), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := make([]runtime.Object, 0)
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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
