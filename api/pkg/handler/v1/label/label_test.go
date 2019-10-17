package label_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/label"
)

func TestListSystemLabels(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name            string
		ExpectedLabels  apiv1.ResourceLabelMap
		HTTPStatus      int
		ExistingAPIUser *apiv1.User
	}{
		// scenario 1
		{
			Name:            "scenario 1: list system labels",
			ExpectedLabels:  label.GetSystemLabels(),
			HTTPStatus:      http.StatusOK,
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/labels/system"), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actual := decodeResourceLabelMap(res.Body, t)
			expected := tc.ExpectedLabels

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("actual map is different that the expected one. Expected: %v, got %v", expected, actual)
			}
		})
	}
}

func decodeResourceLabelMap(r io.Reader, t *testing.T) apiv1.ResourceLabelMap {
	var res apiv1.ResourceLabelMap
	dec := json.NewDecoder(r)
	err := dec.Decode(&res)
	if err != nil {
		t.Fatal(err)
	}

	return res
}
