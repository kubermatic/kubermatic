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

package constrainttemplate_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
)

func TestListConstraintTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                        string
		ExpectedConstraintTemplates []apiv2.ConstraintTemplate
		HTTPStatus                  int
		ExistingAPIUser             *apiv1.User
		ExistingObjects             []runtime.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list all constraint templates",
			ExpectedConstraintTemplates: []apiv2.ConstraintTemplate{
				test.GenDefaultConstraintTemplate("ct1"),
				test.GenDefaultConstraintTemplate("ct2"),
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				genConstraintTemplate("ct1"),
				genConstraintTemplate("ct2"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/constrainttemplates", strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualCTs := test.NewConstraintTemplateV1SliceWrapper{}
			actualCTs.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedCTs := test.NewConstraintTemplateV1SliceWrapper(tc.ExpectedConstraintTemplates)
			wrappedExpectedCTs.Sort()

			actualCTs.EqualOrDie(wrappedExpectedCTs, t)
		})
	}
}

func TestGetConstraintTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		CTName           string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []runtime.Object
	}{
		{
			Name:             "scenario 1: get existing constraint template",
			CTName:           "ct1",
			ExpectedResponse: `{"name":"ct1","spec":{"crd":{"spec":{"names":{"kind":"labelconstraint","shortNames":["lc"]}}},"targets":[{"target":"admission.k8s.gatekeeper.sh","rego":"\n\t\tpackage k8srequiredlabels\n\n        deny[{\"msg\": msg, \"details\": {\"missing_labels\": missing}}] {\n          provided := {label | input.review.object.metadata.labels[label]}\n          required := {label | label := input.parameters.labels[_]}\n          missing := required - provided\n          count(missing) \u003e 0\n          msg := sprintf(\"you must provide labels: %v\", [missing])\n        }"}]},"status":{}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				genConstraintTemplate("ct1"),
				genConstraintTemplate("ct2"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 1: get non-existing constraint template",
			CTName:           "missing",
			ExpectedResponse: `{"error":{"code":404,"message":"constrainttemplates.kubermatic.k8s.io \"missing\" not found"}}`,
			HTTPStatus:       http.StatusNotFound,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				genConstraintTemplate("ct1"),
				genConstraintTemplate("ct2"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/constrainttemplates/%s", tc.CTName), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateConstraintTemplates(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		CTtoCreate       apiv2.ConstraintTemplate
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
	}{
		{
			Name:             "scenario 1: admin can create constraint template",
			CTtoCreate:       test.GenDefaultConstraintTemplate("labelconstraint"),
			ExpectedResponse: `{"name":"labelconstraint","spec":{"crd":{"spec":{"names":{"kind":"labelconstraint","shortNames":["lc"]}}},"targets":[{"target":"admission.k8s.gatekeeper.sh","rego":"\n\t\tpackage k8srequiredlabels\n\n        deny[{\"msg\": msg, \"details\": {\"missing_labels\": missing}}] {\n          provided := {label | input.review.object.metadata.labels[label]}\n          required := {label | label := input.parameters.labels[_]}\n          missing := required - provided\n          count(missing) \u003e 0\n          msg := sprintf(\"you must provide labels: %v\", [missing])\n        }"}]},"status":{}}`,
			HTTPStatus:       http.StatusOK,
			ExistingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
		{
			Name:             "scenario 2: non-admin can not create constraint template",
			CTtoCreate:       test.GenDefaultConstraintTemplate("labelconstraint"),
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 3: admin cannot create invalid constraint template",
			CTtoCreate:       test.GenDefaultConstraintTemplate("invalid"),
			ExpectedResponse: `{"error":{"code":400,"message":"template's name invalid is not equal to the lowercase of CRD's Kind: labelconstraint"}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var reqBody struct {
				Name string                         `json:"name"`
				Spec v1beta1.ConstraintTemplateSpec `json:"spec"`
			}
			reqBody.Spec = tc.CTtoCreate.Spec
			reqBody.Name = tc.CTtoCreate.Name

			body, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("error marshalling body into json: %v", err)
			}
			req := httptest.NewRequest("POST", "/api/v2/constrainttemplates", bytes.NewBuffer(body))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, []runtime.Object{test.APIUserToKubermaticUser(*tc.ExistingAPIUser)}, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func genConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = v1beta1.ConstraintTemplateSpec{
		CRD: v1beta1.CRD{
			Spec: v1beta1.CRDSpec{
				Names: v1beta1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
			},
		},
		Targets: []v1beta1.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Rego: `
		package k8srequiredlabels

        deny[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }`,
			},
		},
	}

	return ct
}
