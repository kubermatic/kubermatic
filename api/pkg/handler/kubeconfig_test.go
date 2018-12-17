package handler

import (
	"encoding/base64"
	"encoding/json"
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
	testDatacenter          = "us-central1"
	testExpectedRedirectURI = "/api/v1/kubeconfig"
	testProjectName         = "my-first-project"
	testClusterName         = "AbcCluster"
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
		ClusterID                 string
		ProjectID                 string
		UserID                    string
		Datacenter                string
		Nonce                     string
		HTTPStatusInitPhase       int
		ExistingKubermaticObjects []runtime.Object
		ExpectedRedirectURI       string
		ExpectedExchangeCodePhase ExpectedKubeconfigResp
	}{
		{
			Name:                      "scenario 1, no parameters for url",
			HTTPStatusInitPhase:       http.StatusInternalServerError,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{},
		},
		{
			Name:                "scenario 2, incorrect user ID in state",
			ClusterID:           testClusterID,
			ProjectID:           testingProjectName,
			UserID:              "0000",
			Datacenter:          testDatacenter,
			HTTPStatusInitPhase: http.StatusNotFound,
		},
		{
			Name:                      "scenario 2, exchange phase error: incorrect state parameter: invalid nonce",
			ClusterID:                 testClusterID,
			ProjectID:                 testingProjectName,
			UserID:                    testUserID,
			Datacenter:                testDatacenter,
			Nonce:                     "abc", // incorrect length
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExpectedRedirectURI:       testExpectedRedirectURI,
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: fmt.Sprintf(`{"error":{"code":400,"message":"incorrect value of state parameter = abc"}}%c`, '\n'),
				HTTPStatus:   http.StatusBadRequest,
			},
		},
		{
			Name:                      "scenario 4, successful scenario",
			ClusterID:                 testClusterID,
			ProjectID:                 testingProjectName,
			UserID:                    testUserID,
			Datacenter:                testDatacenter,
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExpectedRedirectURI:       testExpectedRedirectURI,
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: testKubeconfig,
				HTTPStatus:   http.StatusOK,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			reqURL := fmt.Sprintf("/api/v1/kubeconfig?cluster_id=%s&project_id=%s&user_id=%s&datacenter=%s", tc.ClusterID, tc.ProjectID, tc.UserID, tc.Datacenter)
			req := httptest.NewRequest("GET", reqURL, strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(apiv1.User{}, []runtime.Object{}, tc.ExistingKubermaticObjects, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			assert.Equal(t, tc.HTTPStatusInitPhase, res.Code)

			// Redirection to dex provider
			if res.Code == http.StatusSeeOther {
				location, err := res.Result().Location()
				if err != nil {
					t.Fatalf("expected url for redirection %v", err)
				}

				// validate
				redirectURI := location.Query().Get("redirect_uri")
				assert.Equal(t, tc.ExpectedRedirectURI, redirectURI)

				encodedState, err := url.QueryUnescape(location.Query().Get("state"))
				if err != nil {
					t.Fatalf("incorrect state format %v", err)
				}
				decodeState, err := base64.StdEncoding.DecodeString(encodedState)
				if err != nil {
					t.Fatalf("error decoding state %v", err)
				}

				state, err := unmarshalState(decodeState)
				if err != nil {
					t.Fatalf("error unmarshal state %v", err)
				}

				// validate
				assert.Equal(t, tc.ClusterID, state.ClusterID)
				assert.Equal(t, tc.Datacenter, state.Datacenter)
				assert.Equal(t, tc.ProjectID, state.ProjectID)
				assert.Equal(t, tc.UserID, state.UserID)

				// copy generated nonce to cookie
				cookieValue := state.Nonce

				// override the Nonce if test scenario set the value
				// if not use generated by server
				if tc.Nonce != "" {
					state.Nonce = tc.Nonce
				}

				encodedState, err = marshalEncodeState(state)
				if err != nil {
					t.Fatalf("error marshal state %v", err)
				}
				urlExchangeCodePhase := fmt.Sprintf("/api/v1/kubeconfig?code=%s&state=%s", testAuthorizationCode, encodedState)

				// call kubeconfig endpoint after authentication
				// exchange code phase
				req = httptest.NewRequest("GET", urlExchangeCodePhase, strings.NewReader(""))
				res = httptest.NewRecorder()

				// create secure cookie
				if encoded, err := secureCookie.Encode(csrfCookieName, cookieValue); err == nil {
					// Drop a cookie on the recorder.
					http.SetCookie(res, &http.Cookie{Name: "csrf_token", Value: encoded})

					// Copy the Cookie over to a new Request
					req.Header.Add("Cookie", res.HeaderMap["Set-Cookie"][0])
				}

				// act
				ep.ServeHTTP(res, req)

				// validate
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

func marshalEncodeState(oidcState state) (string, error) {

	rawState, err := json.Marshal(oidcState)
	if err != nil {
		return "", err
	}
	encodedState := base64.StdEncoding.EncodeToString(rawState)
	return url.QueryEscape(encodedState), nil

}

func unmarshalState(rawState []byte) (state, error) {
	oidcState := state{}
	if err := json.Unmarshal(rawState, &oidcState); err != nil {
		return state{}, err
	}
	return oidcState, nil
}
