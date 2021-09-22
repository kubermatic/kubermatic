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

package admissionplugin_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/semver"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()

	v114, err := semver.NewSemver("1.14")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		name             string
		version          string
		plugins          []ctrlruntimeclient.Object
		httpStatus       int
		expectedResponse string
	}{
		{
			name:    "test get default plugins",
			version: "1.13",
			plugins: []ctrlruntimeclient.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "FirstPlugin",
						FromVersion: v114,
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `["PodNodeSelector","PodSecurityPolicy"]`,
		},
		{
			name:    "test get plugins for version 1.14",
			version: "1.14.5",
			plugins: []ctrlruntimeclient.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName:  "FirstPlugin",
						FromVersion: v114,
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `["FirstPlugin","PodNodeSelector","PodSecurityPolicy"]`,
		},
		{
			name:    "test get plugins for all versions",
			version: "1.13.5",
			plugins: []ctrlruntimeclient.Object{
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: metav1.ObjectMeta{
						Name: "first",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "FirstPlugin",
					},
				},
				&kubermaticv1.AdmissionPlugin{
					ObjectMeta: metav1.ObjectMeta{
						Name: "second",
					},
					Spec: kubermaticv1.AdmissionPluginSpec{
						PluginName: "SecondPlugin",
					},
				},
			},
			httpStatus:       http.StatusOK,
			expectedResponse: `["FirstPlugin","PodNodeSelector","PodSecurityPolicy","SecondPlugin"]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admission/plugins/%s", tc.version), strings.NewReader(""))
			res := httptest.NewRecorder()

			apiUser := test.GenDefaultAPIUser()
			router, err := test.CreateTestEndpoint(*apiUser, nil, tc.plugins, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v\n", err)
			}
			router.ServeHTTP(res, req)

			// validate
			assert.Equal(t, tc.httpStatus, res.Code)

			if res.Code == http.StatusOK {
				compareJSON(t, res, tc.expectedResponse)
			}
		})
	}
}

func compareJSON(t *testing.T, res *httptest.ResponseRecorder, expectedResponseString string) {
	t.Helper()
	var actualResponse interface{}
	var expectedResponse interface{}

	// var err error
	bBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("Unable to read response body")
	}
	err = json.Unmarshal(bBytes, &actualResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(expectedResponseString), &expectedResponse)
	if err != nil {
		t.Fatalf("Error marshaling string 2 :: %s", err.Error())
	}
	if !equality.Semantic.DeepEqual(actualResponse, expectedResponse) {
		t.Fatalf("Objects are different: %v", diff.ObjectDiff(actualResponse, expectedResponse))
	}
}
