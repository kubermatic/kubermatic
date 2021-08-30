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

package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-test/deep"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKubermaticVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	res := httptest.NewRecorder()
	ep, err := test.CreateTestEndpoint(*test.GenDefaultAPIUser(), []ctrlruntimeclient.Object{}, nil, nil, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create testStruct endpoint due to %v", err)
	}
	ep.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
	}

	var gotVersion apiv1.KubermaticVersions
	err = json.Unmarshal(res.Body.Bytes(), &gotVersion)
	if err != nil {
		t.Fatal(err)
	}

	expectedVersions := apiv1.KubermaticVersions{
		API: kubermatic.NewFakeVersions().Kubermatic,
	}

	if diff := deep.Equal(gotVersion, expectedVersions); diff != nil {
		t.Fatalf("got different upgrade response than expected. Diff: %v", diff)
	}
}
