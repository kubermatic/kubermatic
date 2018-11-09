package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	testExpectedRedirectURI  = "/api/v1/kubeconfig"
	testProjectName          = "my-first-project"
	testClusterName          = "AbcCluster"
	testCorrectState         = `{"nonce":"nonce=TODO","cluster_id":"AbcClusterID","project_id":"my-first-project-ID","user_id":"1233","datacenter":"us-central1"}`
	testMissingNonceState    = `{"cluster_id":"AbcClusterID","project_id":"my-first-project-ID","user_id":"1233","datacenter":"us-central1"}`
	testIncorrectUserIDState = `{"nonce":"nonce=TODO","cluster_id":"AbcClusterID","project_id":"my-first-project-ID","user_id":"0000","datacenter":"us-central1"}`
)

const testKubeconfig = `apiVersion: v1
clusters:
- cluster:
    server: test.fake.io
  name: AbcClusterID
contexts:
- context:
    cluster: AbcClusterID
    user: john@acme.com
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: john@acme.com
  user:
    auth-provider:
      config:
        client-id: kubermatic
        client-secret: secret
        id-token: fakeTokenId
        idp-issuer-url: url://dex
        refresh-token: fakeRefreshToken
      name: oidc
`

type ExpectedKubeconfigResp struct {
	BodyResponse string
	HTTPStatus   int
}

func TestCreateOIDCKubeconfig(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                      string
		Body                      string
		URLArgInitPhase           string
		URLArgExchangeCodePhase   string
		HTTPStatusInitPhase       int
		ExistingKubermaticObjects []runtime.Object
		ExistingAPIUser           apiv1.LegacyUser
		ExpectedRedirectURI       string
		ExpectedState             string
		ExpectedExchangeCodePhase ExpectedKubeconfigResp
	}{
		{
			Name:                      "scenario 1, no parameters for url",
			Body:                      "",
			URLArgInitPhase:           "/api/v1/kubeconfig",
			HTTPStatusInitPhase:       http.StatusInternalServerError,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingAPIUser:           apiv1.LegacyUser{},
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{},
		},
		{
			Name:                      "scenario 2, exchange phase error: incorrect state parameter, nonce missing",
			Body:                      ``,
			URLArgInitPhase:           fmt.Sprintf("/api/v1/kubeconfig?cluster_id=%s&project_id=%s&user_id=%s&datacenter=us-central1", testClusterID, testingProjectName, testUserID),
			URLArgExchangeCodePhase:   fmt.Sprintf("/api/v1/kubeconfig?code=%s&state=%s", testAuthorizationCode, genTestKubeconfigState(testMissingNonceState)),
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingAPIUser: apiv1.LegacyUser{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExpectedRedirectURI: testExpectedRedirectURI,
			ExpectedState:       testCorrectState,
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: fmt.Sprintf(`{"error":{"code":400,"message":"incorrect value of state parameter = "}}%c`, '\n'),
				HTTPStatus:   http.StatusBadRequest,
			},
		},
		{
			Name:                      "scenario 3, exchange phase error: incorrect user ID in state",
			Body:                      ``,
			URLArgInitPhase:           fmt.Sprintf("/api/v1/kubeconfig?cluster_id=%s&project_id=%s&user_id=%s&datacenter=us-central1", testClusterID, testingProjectName, testUserID),
			URLArgExchangeCodePhase:   fmt.Sprintf("/api/v1/kubeconfig?code=%s&state=%s", testAuthorizationCode, genTestKubeconfigState(testIncorrectUserIDState)),
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingAPIUser: apiv1.LegacyUser{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExpectedRedirectURI: testExpectedRedirectURI,
			ExpectedState:       testCorrectState,
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: fmt.Sprintf(`{"error":{"code":404,"message":"users.kubermatic.k8s.io \"0000\" not found"}}%c`, '\n'),
				HTTPStatus:   http.StatusNotFound,
			},
		},
		{
			Name:                      "scenario 4, successful scenario",
			Body:                      ``,
			URLArgInitPhase:           fmt.Sprintf("/api/v1/kubeconfig?cluster_id=%s&project_id=%s&user_id=%s&datacenter=us-central1", testClusterID, testingProjectName, testUserID),
			URLArgExchangeCodePhase:   fmt.Sprintf("/api/v1/kubeconfig?code=%s&state=%s", testAuthorizationCode, genTestKubeconfigState(testCorrectState)),
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingAPIUser: apiv1.LegacyUser{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExpectedRedirectURI: testExpectedRedirectURI,
			ExpectedState:       testCorrectState,
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: testKubeconfig,
				HTTPStatus:   http.StatusOK,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.URLArgInitPhase, strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(tc.ExistingAPIUser, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// valdiate
			assert.Equal(t, tc.HTTPStatusInitPhase, res.Code)

			// Redirection to dex provider
			if res.Code == http.StatusSeeOther {
				location, err := res.Result().Location()
				if err != nil {
					t.Fatalf("expected url for redirection %v", err)
				}

				// valdiate
				redirectURI := location.Query().Get("redirect_uri")
				assert.Equal(t, tc.ExpectedRedirectURI, redirectURI)

				state, err := url.QueryUnescape(location.Query().Get("state"))
				if err != nil {
					t.Fatalf("incorrect state format %v", err)
				}
				decodeState, err := base64.StdEncoding.DecodeString(state)
				if err != nil {
					t.Fatalf("error decoding state %v", err)
				}

				// valdiate
				assert.Equal(t, tc.ExpectedState, string(decodeState))

				// call kubeconfig endpoint after authentication
				// exchange code phase
				req = httptest.NewRequest("GET", tc.URLArgExchangeCodePhase, strings.NewReader(tc.Body))
				res = httptest.NewRecorder()
				// act
				ep.ServeHTTP(res, req)

				// valdiate
				assert.Equal(t, tc.ExpectedExchangeCodePhase.HTTPStatus, res.Code)

				// validate
				assert.Equal(t, tc.ExpectedExchangeCodePhase.BodyResponse, res.Body.String())

			}
		})
	}
}

func genTestKubeconfigKubermaticObjects() []runtime.Object {
	return []runtime.Object{
		// add some project
		genProject(testProjectName, kubermaticapiv1.ProjectActive, defaultCreationTimestamp()),
		// add John
		genUser(testUserID, testUserName, testUserEmail),
		// make John the owner of the first project and the editor of the second
		genBinding(testingProjectName, testUserEmail, "owners"),
		// add a cluster
		genCluster(testClusterID, testClusterName, genDefaultProject().Name, defaultCreationTimestamp()),
	}
}

func genTestKubeconfigState(stateString string) string {
	encodedState := base64.StdEncoding.EncodeToString([]byte(stateString))
	return url.QueryEscape(encodedState)
}
