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

	"k8s.io/apimachinery/pkg/runtime"
)

func TestListConstraints(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                string
		ProjectID           string
		ClusterID           string
		HTTPStatus          int
		ExistingAPIUser     *apiv1.User
		ExistingObjects     []runtime.Object
		ExpectedConstraints []apiv2.Constraint
	}{
		{
			Name:      "scenario 1: user can list accessible constraint",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{
				genDefaultConstraint("ct1"),
				genDefaultConstraint("ct2"),
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		{
			Name:                "scenario 2: unauthorized user can not get constraints",
			ProjectID:           test.GenDefaultProject().Name,
			ClusterID:           test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{},
			HTTPStatus:          http.StatusForbidden,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:      "scenario 3: admin user list all constraints",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			ExpectedConstraints: []apiv2.Constraint{
				genDefaultConstraint("ct1"),
				genDefaultConstraint("ct2"),
			},
			HTTPStatus: http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/constraints",
				tc.ProjectID, tc.ClusterID), strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
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
	t.Parallel()
	testcases := []struct {
		Name             string
		ConstraintName   string
		ProjectID        string
		ClusterID        string
		ExpectedResponse string
		HTTPStatus       int
		ExistingAPIUser  *apiv1.User
		ExistingObjects  []runtime.Object
	}{
		{
			Name:             "scenario 1: user can get accessible constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"requiredlabels","match":{"kinds":[{"kinds":"namespace"}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"rawJSON":"{\"labels\":[ \"gatekeeper\", \"opa\"]}"}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
			),
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
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		{
			Name:             "scenario 3: admin user can get any constraint",
			ConstraintName:   "ct1",
			ProjectID:        test.GenDefaultProject().Name,
			ClusterID:        test.GenDefaultCluster().Name,
			ExpectedResponse: `{"name":"ct1","spec":{"constraintType":"requiredlabels","match":{"kinds":[{"kinds":"namespace"}],"labelSelector":{},"namespaceSelector":{}},"parameters":{"rawJSON":"{\"labels\":[ \"gatekeeper\", \"opa\"]}"}}}`,
			HTTPStatus:       http.StatusOK,
			ExistingObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				genConstraint("ct1", test.GenDefaultCluster().Status.NamespaceName),
				genConstraint("ct2", test.GenDefaultCluster().Status.NamespaceName),
				genKubermaticUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/kubernetes/clusters/%s/constraints/%s",
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

func genConstraint(name, namespace string) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Kind = kubermaticv1.ConstraintKind
	ct.APIVersion = kubermaticv1.SchemeGroupVersion.String()
	ct.Name = name
	ct.Namespace = namespace
	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: "requiredlabels",
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{Kinds: "namespace", APIGroups: ""},
			},
		},
		Parameters: kubermaticv1.Parameters{
			RawJSON: `{"labels":[ "gatekeeper", "opa"]}`,
		},
	}

	return ct
}

func genKubermaticUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func genDefaultConstraint(name string) apiv2.Constraint {
	return apiv2.Constraint{
		Name: name,
		Spec: kubermaticv1.ConstraintSpec{
			ConstraintType: "requiredlabels",
			Match: kubermaticv1.Match{
				Kinds: []kubermaticv1.Kind{
					{Kinds: "namespace", APIGroups: ""},
				},
			},
			Parameters: kubermaticv1.Parameters{
				RawJSON: `{"labels":[ "gatekeeper", "opa"]}`,
			},
		},
	}
}
