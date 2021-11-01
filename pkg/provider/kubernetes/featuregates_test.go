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

package kubernetes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestFeatureGatesEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          features.FeatureGate
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:                      "feature gates status",
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:    http.StatusOK,
			ExpectedResponse: features.FeatureGate{
				"feature-gate-test":         true,
				"feature-gate-test-another": false,
			},
		},
	}

	dummyKubermaticConfiguration := operatorv1alpha1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: test.KubermaticNamespace,
		},
		Spec: operatorv1alpha1.KubermaticConfigurationSpec{
			Versions: operatorv1alpha1.KubermaticVersionsConfiguration{
				Kubernetes: operatorv1alpha1.KubermaticVersioningConfiguration{
					Versions: test.GenDefaultVersions(),
				},
			},
			FeatureGates: map[string]sets.Empty{
				"feature-gate-test=true":          {},
				"feature-gate-test-another=false": {},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v2/featuregates", nil)
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, &dummyKubermaticConfiguration, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				var featureGates features.FeatureGate
				if err := json.Unmarshal(resp.Body.Bytes(), &featureGates); err != nil {
					t.Fatalf("failed to unmarshal response due to: %v", err)
				}

				if len(tc.ExpectedResponse) != len(featureGates) {
					t.Fatalf("expected %d feature gates, got %d", len(tc.ExpectedResponse), len(featureGates))
				}

				for key, val := range tc.ExpectedResponse {
					v, ok := featureGates[key]
					if !ok || v != val {
						t.Fatalf("expected %s to be %t, got %t", key, val, v)
					}
				}
			}
		})
	}
}
