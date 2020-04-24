package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetGlobalSettings(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: user gets settings first time",
			expectedResponse:       `{"customLinks":[],"cleanupOptions":{"Enabled":false,"Enforced":false},"defaultNodeCount":10,"clusterTypeOptions":0,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":false,"enableDashboard":true,"enableOIDCKubeconfig":false}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:             "scenario 2: user gets existing global settings",
			expectedResponse: `{"customLinks":[{"label":"label","url":"url:label","icon":"icon","location":"EU"}],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":5,"clusterTypeOptions":5,"displayDemoInfo":true,"displayAPIDocs":true,"displayTermsOfService":true,"enableDashboard":false,"enableOIDCKubeconfig":false}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				genDefaultGlobalSettings()},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("GET", "/api/v1/admin/settings", strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func TestUpdateGlobalSettings(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		body                   string
		expectedResponse       string
		httpStatus             int
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			name:                   "scenario 1: unauthorized user updates settings",
			body:                   `{"customLinks":[{"label":"label","url":"url:label","icon":"icon","location":"EU"}],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":100,"clusterTypeOptions":20,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":true}`,
			expectedResponse:       `{"error":{"code":403,"message":"forbidden: \"bob@acme.com\" doesn't have admin rights"}}`,
			httpStatus:             http.StatusForbidden,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			name:                   "scenario 2: authorized user updates default settings",
			body:                   `{"customLinks":[{"label":"label","url":"url:label","icon":"icon","location":"EU"}],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":100,"clusterTypeOptions":20,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":true}`,
			expectedResponse:       `{"customLinks":[{"label":"label","url":"url:label","icon":"icon","location":"EU"}],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":100,"clusterTypeOptions":20,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":true,"enableDashboard":true,"enableOIDCKubeconfig":false}`,
			httpStatus:             http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true)},
			existingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			name:             "scenario 3: authorized user updates existing global settings",
			body:             `{"customLinks":[],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":100,"clusterTypeOptions":20,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":true}`,
			expectedResponse: `{"customLinks":[],"cleanupOptions":{"Enabled":true,"Enforced":true},"defaultNodeCount":100,"clusterTypeOptions":20,"displayDemoInfo":false,"displayAPIDocs":false,"displayTermsOfService":true,"enableDashboard":false,"enableOIDCKubeconfig":false}`,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: []runtime.Object{genUser("Bob", "bob@acme.com", true),
				genDefaultGlobalSettings()},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			req := httptest.NewRequest("PATCH", "/api/v1/admin/settings", strings.NewReader(tc.body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
}

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}

func genDefaultGlobalSettings() *kubermaticv1.KubermaticSetting {
	return &kubermaticv1.KubermaticSetting{
		ObjectMeta: v1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			CustomLinks: []kubermaticv1.CustomLink{
				{
					Label:    "label",
					URL:      "url:label",
					Icon:     "icon",
					Location: "EU",
				},
			},
			CleanupOptions: kubermaticv1.CleanupOptions{
				Enabled:  true,
				Enforced: true,
			},
			DefaultNodeCount:      5,
			ClusterTypeOptions:    5,
			DisplayDemoInfo:       true,
			DisplayAPIDocs:        true,
			DisplayTermsOfService: true,
		},
	}
}
