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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
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
		ExistingKObjs               []runtime.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list all constraint templates",
			ExpectedConstraintTemplates: []apiv2.ConstraintTemplate{
				test.GenDefaultConstraintTemplate("ct1"),
				test.GenDefaultConstraintTemplate("ct2"),
			},
			HTTPStatus: http.StatusOK,
			ExistingKObjs: test.GenDefaultKubermaticObjects(
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
			ep, clientSet, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, nil, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			for _, obj := range tc.ExistingKObjs {
				err := clientSet.FakeClient.Create(context.Background(), obj)
				if err != nil {
					t.Fatalf("failed to create existing objects due to %v", err)
				}
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualCTs := test.NewConstraintTemplateV1SliceWrapper{}
			actualCTs.DecodeOrDie(res.Body, t)

			wrappedExpectedCTs := test.NewConstraintTemplateV1SliceWrapper(tc.ExpectedConstraintTemplates)
			//wrappedExpectedCTs.Sort()

			actualCTs.EqualOrDie(wrappedExpectedCTs, t)
		})
	}
}

func genConstraintTemplate(name string) *v1beta1.ConstraintTemplate {
	ct := &v1beta1.ConstraintTemplate{}
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
