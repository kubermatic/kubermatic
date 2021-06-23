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

package constraint_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/handler/v2/constraint"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubermaticNamespace = "kubermatic"
)

func TestListConstraints(t *testing.T) {
	if err := test.RegisterScheme(test.SchemeBuilder); err != nil {
		t.Fatal(err)
	}

	t.Parallel()
	testcases := []struct {
		Name                      string
		ProjectID                 string
		ClusterID                 string
		HTTPStatus                int
		ExistingAPIUser           *apiv1.User
		ExistingObjects           []ctrlruntimeclient.Object
		ExistingGatekeeperObjects []ctrlruntimeclient.Object
		ExpectedConstraints       []apiv2.Constraint
	}{
		{
			Name:      "scenario 1: user can list accessible constraint",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{
				test.GenDefaultAPIConstraint("ct1", "RequiredLabel"),
				test.GenDefaultAPIConstraint("ct2", "RequiredLabel"),
				test.GenDefaultAPIConstraint("ct3", "UniqueLabel"),
				func() apiv2.Constraint {
					c := test.GenDefaultAPIConstraint("ct4", "UniqueLabel")
					c.Status = &apiv2.ConstraintStatus{Synced: pointer.BoolPtr(false)}
					return c
				}(),
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				test.GenConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				test.GenConstraint("ct3", test.GenDefaultCluster().Status.NamespaceName, "UniqueLabel"),
				test.GenConstraint("ct4", test.GenDefaultCluster().Status.NamespaceName, "UniqueLabel"),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
				genGatekeeperConstraint("ct2", "RequiredLabel", t),
				genGatekeeperConstraint("ct3", "UniqueLabel", t),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:                "scenario 2: unauthorized user can not get constraints",
			ProjectID:           test.GenDefaultProject().Name,
			ClusterID:           test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{},
			HTTPStatus:          http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				test.GenConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
				genGatekeeperConstraint("ct2", "RequiredLabel", t),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:      "scenario 3: admin user list all constraints",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{
				test.GenDefaultAPIConstraint("ct1", "RequiredLabel"),
				test.GenDefaultAPIConstraint("ct2", "RequiredLabel"),
				test.GenDefaultAPIConstraint("ct3", "UniqueLabel"),
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				test.GenConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				test.GenConstraint("ct3", test.GenDefaultCluster().Status.NamespaceName, "UniqueLabel"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
				genGatekeeperConstraint("ct2", "RequiredLabel", t),
				genGatekeeperConstraint("ct3", "UniqueLabel", t),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/constraints",
				tc.ProjectID, tc.ClusterID), strings.NewReader(""))
			res := httptest.NewRecorder()
			ctx := context.Background()

			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			for _, gkObject := range tc.ExistingGatekeeperObjects {
				err = clientsSets.FakeClient.Create(ctx, gkObject)
				if err != nil {
					t.Fatalf("failed to create gk object %v due to %v", gkObject, err)
				}
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			if res.Code != http.StatusOK {
				return
			}

			actualCTs := test.NewConstraintsSliceWrapper{}
			actualCTs.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedCTs := test.NewConstraintsSliceWrapper(tc.ExpectedConstraints)
			wrappedExpectedCTs.Sort()

			actualCTs.EqualOrDie(wrappedExpectedCTs, t)
		})
	}
}

func TestGetConstraints(t *testing.T) {
	if err := test.RegisterScheme(test.SchemeBuilder); err != nil {
		t.Fatal(err)
	}

	t.Parallel()
	testcases := []struct {
		Name                      string
		ConstraintName            string
		ProjectID                 string
		ClusterID                 string
		ExpectedResponse          string
		HTTPStatus                int
		ExistingAPIUser           *apiv1.User
		ExistingObjects           []ctrlruntimeclient.Object
		ExistingGatekeeperObjects []ctrlruntimeclient.Object
	}{
		{
			Name:             "scenario 1: user can get accessible constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}},"status":{"enforcement":"true","auditTimestamp":"2019-05-11T01:46:13Z","violations":[{"enforcementAction":"deny","kind":"Namespace","message":"'you must provide labels: {\"gatekeeper\"}'","name":"default"},{"enforcementAction":"deny","kind":"Namespace","message":"'you must provide labels: {\"gatekeeper\"}'","name":"gatekeeper"}],"synced":true}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: unauthorized user can not get constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: admin user can get any constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}},"status":{"enforcement":"true","auditTimestamp":"2019-05-11T01:46:13Z","violations":[{"enforcementAction":"deny","kind":"Namespace","message":"'you must provide labels: {\"gatekeeper\"}'","name":"default"},{"enforcementAction":"deny","kind":"Namespace","message":"'you must provide labels: {\"gatekeeper\"}'","name":"gatekeeper"}],"synced":true}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{
				genGatekeeperConstraint("ct1", "RequiredLabel", t),
			},
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 4: user can get constraint which is not synced yet on the user cluster",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}},"status":{"synced":false}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingGatekeeperObjects: []ctrlruntimeclient.Object{},
			ExistingAPIUser:           test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/constraints/%s",
				tc.ProjectID, tc.ClusterID, tc.ConstraintName), strings.NewReader(""))
			res := httptest.NewRecorder()
			ctx := context.Background()

			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, nil, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			for _, gkObject := range tc.ExistingGatekeeperObjects {
				err = clientsSets.FakeClient.Create(ctx, gkObject)
				if err != nil {
					t.Fatalf("failed to create gk object due to %v", err)
				}
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func genGatekeeperConstraint(name, kind string, t *testing.T) *unstructured.Unstructured {
	ct := &unstructured.Unstructured{}
	ct.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   constraint.ConstraintsGroup,
		Version: constraint.ConstraintsVersion,
		Kind:    kind,
	})
	ct.SetName(name)
	ct.SetNamespace(constraint.ConstraintNamespace)

	// set spec
	spec := &kubermaticv1.ConstraintSpec{
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{Kinds: []string{"namespace"}, APIGroups: []string{""}},
			},
		},
	}

	specMap, err := unmarshallToJSONMap(spec)
	if err != nil {
		t.Fatal(err)
	}
	err = unstructured.SetNestedField(ct.Object, specMap, "spec")
	if err != nil {
		t.Fatalf("error setting constraint spec field: %v", err)
	}

	// Set parameters
	parameters := map[string]interface{}{
		"labels": []interface{}{"gatekeeper", "opa"},
	}

	err = unstructured.SetNestedField(ct.Object, parameters, "spec", "parameters")
	if err != nil {
		t.Fatalf("error setting constraint parameters field: %v", err)
	}

	// set status
	apiStatus := apiv2.ConstraintStatus{
		Enforcement:    "true",
		AuditTimestamp: "2019-05-11T01:46:13Z",
		Violations: []apiv2.Violation{
			{
				EnforcementAction: "deny",
				Kind:              "Namespace",
				Message:           "'you must provide labels: {\"gatekeeper\"}'",
				Name:              "default",
			},
			{
				EnforcementAction: "deny",
				Kind:              "Namespace",
				Message:           "'you must provide labels: {\"gatekeeper\"}'",
				Name:              "gatekeeper",
			},
		},
	}
	statusMap, err := unmarshallToJSONMap(apiStatus)
	if err != nil {
		t.Fatal(err)
	}
	err = unstructured.SetNestedField(ct.Object, statusMap, "status")
	if err != nil {
		t.Fatalf("error setting constraint status field: %v", err)
	}

	return ct
}

func unmarshallToJSONMap(object interface{}) (map[string]interface{}, error) {

	raw, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("error marshalling: %v", err)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %v", err)
	}

	return result, nil
}

func TestDeleteConstraints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		ConstraintName   string
		ProjectID        string
		ClusterID        string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []ctrlruntimeclient.Object
	}{
		{
			Name:             "scenario 1: user can delete constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: unauthorized user can not delete constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: admin user can delete any constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/constraints/%s",
				tc.ProjectID, tc.ClusterID, tc.ConstraintName), strings.NewReader(""))
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

func TestCreateConstraints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		Constraint       apiv2.Constraint
		ProjectID        string
		ClusterID        string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []ctrlruntimeclient.Object
	}{
		{
			Name: "scenario 1: user can create constraint",
			Constraint: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel").Spec,
			},
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name: "scenario 2: unauthorized user can not create constraint",
			Constraint: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel").Spec,
			},
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name: "scenario 3: admin user can create constraint in any project/cluster",
			Constraint: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel").Spec,
			},
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name: "scenario 4: cannot create constraint with not existing constraint template",
			Constraint: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel").Spec,
			},
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":400,"message":"Validation failed, constraint needs to have an existing constraint template: constrainttemplates.kubermatic.k8s.io \"requiredlabel\" not found"}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name: "scenario 5: cannot create constraint with invalid parameters",
			Constraint: func() apiv2.Constraint {
				c := test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel")
				c.Spec.Parameters = map[string]interface{}{"labels": "fail"}
				return apiv2.Constraint{
					Name: "ct1",
					Spec: c.Spec,
				}
			}(),
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":400,"message":"Validation failed, constraint spec is not valid: spec.parameters.labels: Invalid value: \"string\": labels in body must be of type array: \"string\""}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name: "scenario 6: cannot create rawJSON constraint with invalid parameters",
			Constraint: func() apiv2.Constraint {
				c := test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel")
				c.Spec.Parameters = kubermaticv1.Parameters{
					"rawJSON": `{"labels":"gatekeeper"}`,
				}
				return apiv2.Constraint{
					Name: "ct1",
					Spec: c.Spec,
				}
			}(),
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"error":{"code":400,"message":"Validation failed, constraint spec is not valid: spec.parameters.labels: Invalid value: \"string\": labels in body must be of type array: \"string\""}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		var reqBody struct {
			Name string                      `json:"name"`
			Spec kubermaticv1.ConstraintSpec `json:"spec"`
		}
		reqBody.Spec = tc.Constraint.Spec
		reqBody.Name = tc.Constraint.Name

		body, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("error marshalling body into json: %v", err)
		}
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/constraints",
				tc.ProjectID, tc.ClusterID), bytes.NewBuffer(body))
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

func TestPatchConstraints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		ConstraintName   string
		ProjectID        string
		ClusterID        string
		Patch            string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []ctrlruntimeclient.Object
	}{
		{
			Name:             "scenario 1: user can patch constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			Patch:            `{"spec":{"constraintType":"somethingdifferentthatshouldnotbeapplied","match":{"kinds":[{"kinds":["pods"], "apiGroups":["v1"]}, {"kinds":["namespaces"]}]}}}`,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["pods"],"apiGroups":["v1"]},{"kinds":["namespaces"]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 2: unauthorized user can not patch constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			Patch:            `{"spec":{"constraintType":"somethingdifferentthatshouldnotbeapplied","match":{"kinds":[{"kinds":["pods"], "apiGroups":["v1"]}, {"kinds":"namespaces"}]}}}`,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: admin user can patch any constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			Patch:            `{"spec":{"constraintType":"somethingdifferentthatshouldnotbeapplied","match":{"kinds":[{"kinds":["pods"], "apiGroups":["v1"]}, {"kinds":["namespaces"]}]}}}`,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["pods"],"apiGroups":["v1"]},{"kinds":["namespaces"]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 4: cannot patch invalid constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			Patch:            `{"spec":{"parameters":{"labels":"gatekeeper"}}}`,
			ExpectedResponse: `{"error":{"code":400,"message":"Validation failed, constraint spec is not valid: spec.parameters.labels: Invalid value: \"string\": labels in body must be of type array: \"string\""}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				test.GenConstraintTemplate("requiredlabel"),
				test.GenConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName, "RequiredLabel"),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/constraints/%s",
				tc.ProjectID, tc.ClusterID, tc.ConstraintName), strings.NewReader(tc.Patch))
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

func genKubermaticUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func TestCreateDefaultConstraints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name             string
		CTtoCreate       apiv2.Constraint
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []ctrlruntimeclient.Object
	}{
		{
			Name: "scenario 1: admin can create default constraint",
			CTtoCreate: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", kubermaticNamespace, "RequiredLabel").Spec,
			},
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"RequiredLabel","active":true,"match":{"kinds":[{"kinds":["namespace"],"apiGroups":[""]}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"labels":["gatekeeper","opa"]},"selector":{"providers":["aws","gcp"],"labelSelector":{"matchLabels":{"deployment":"prod","domain":"sales"},"matchExpressions":[{"key":"cluster","operator":"Exists"}]}}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingAPIUser:  test.GenDefaultAdminAPIUser(),
			ExistingObjects:  []ctrlruntimeclient.Object{test.GenConstraintTemplate("requiredlabel")},
		},
		{
			Name: "scenario 2: non-admin can not create default constraint",
			CTtoCreate: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", kubermaticNamespace, "RequiredLabel").Spec,
			},
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingObjects:  []ctrlruntimeclient.Object{test.GenConstraintTemplate("requiredlabel")},
		},
		{
			Name: "scenario 3: cannot create constraint with not existing constraint template",
			CTtoCreate: apiv2.Constraint{
				Name: "ct1",
				Spec: test.GenConstraint("ct1", kubermaticNamespace, "RequiredLabel").Spec,
			},
			ExpectedResponse: `{"error":{"code":400,"message":"Validation failed, constraint needs to have an existing constraint template: constrainttemplates.kubermatic.k8s.io \"requiredlabel\" not found"}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingAPIUser:  test.GenDefaultAdminAPIUser(),
		},
	}
	for _, tc := range testcases {
		var reqBody struct {
			Name string                      `json:"name"`
			Spec kubermaticv1.ConstraintSpec `json:"spec"`
		}
		reqBody.Spec = tc.CTtoCreate.Spec
		reqBody.Name = tc.CTtoCreate.Name

		body, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("error marshalling body into json: %v", err)
		}
		t.Run(tc.Name, func(t *testing.T) {

			tc.ExistingObjects = append(tc.ExistingObjects, test.APIUserToKubermaticUser(*tc.ExistingAPIUser))

			req := httptest.NewRequest("POST", "/api/v2/constraints", bytes.NewBuffer(body))
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
