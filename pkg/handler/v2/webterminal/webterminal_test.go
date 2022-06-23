/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package webterminal_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testExpectedRedirectURI = ":///api/v2/kubeconfig/secret"
	testClusterName         = "AbcCluster"

	csrfCookieName = "csrf_token"
)

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
		Nonce                     string
		HTTPStatusInitPhase       int
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingObjects           []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedRedirectURI       string
		ExpectedExchangeCodePhase ExpectedKubeconfigResp
	}{
		{
			Name:                      "scenario 1, no parameters for url",
			HTTPStatusInitPhase:       http.StatusInternalServerError,
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{},
		},
		{
			Name:                "scenario 2, incorrect user ID in state",
			ClusterID:           test.ClusterID,
			ProjectID:           test.GenDefaultProject().Name,
			UserID:              "0000",
			ExistingAPIUser:     test.GenDefaultAPIUser(),
			HTTPStatusInitPhase: http.StatusNotFound,
		},
		{
			Name:                      "scenario 3, exchange phase error: incorrect state parameter: invalid nonce",
			ClusterID:                 test.ClusterID,
			ProjectID:                 test.GenDefaultProject().Name,
			UserID:                    test.GenDefaultUser().Name,
			Nonce:                     "abc", // incorrect length
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExpectedRedirectURI:       testExpectedRedirectURI,
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: fmt.Sprintf(`{"error":{"code":400,"message":"incorrect value of state parameter: abc"}}%c`, '\n'),
				HTTPStatus:   http.StatusBadRequest,
			},
		},
		{
			Name:                      "scenario 4, successful scenario",
			ClusterID:                 test.ClusterID,
			ProjectID:                 test.GenDefaultProject().Name,
			UserID:                    test.GenDefaultUser().Name,
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: kubernetes.NamespaceName(test.ClusterID),
						Name:      "admin-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig(test.ClusterID, test.IDToken)),
					},
				},
			},
			ExpectedRedirectURI: testExpectedRedirectURI,
			ExistingAPIUser:     test.GenDefaultAPIUser(),
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: "",
				HTTPStatus:   http.StatusOK,
			},
		},
		{
			Name:                      "scenario 5, the admin can get kubeconfig for Bob cluster",
			ClusterID:                 test.ClusterID,
			ProjectID:                 test.GenDefaultProject().Name,
			UserID:                    test.GenAdminUser("john", "john@acme.com", true).Name,
			HTTPStatusInitPhase:       http.StatusSeeOther,
			ExistingKubermaticObjects: genTestKubeconfigKubermaticObjects(),
			ExistingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: kubernetes.NamespaceName(test.ClusterID),
						Name:      "admin-kubeconfig",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(test.GenerateTestKubeconfig(test.ClusterID, test.IDToken)),
					},
				},
			},
			ExpectedRedirectURI: testExpectedRedirectURI,
			ExistingAPIUser:     test.GenAPIUser("john", "john@acme.com"),
			ExpectedExchangeCodePhase: ExpectedKubeconfigResp{
				BodyResponse: "",
				HTTPStatus:   http.StatusOK,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			reqURL := fmt.Sprintf("/api/v2/kubeconfig/secret?cluster_id=%s&project_id=%s&user_id=%s", tc.ClusterID, tc.ProjectID, tc.UserID)
			req := httptest.NewRequest("GET", reqURL, strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, tc.ExistingObjects, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}

			// act
			ep.ServeHTTP(res, req)
			result := res.Result()
			defer result.Body.Close()

			// validate
			assert.Equal(t, tc.HTTPStatusInitPhase, res.Code)

			// Redirection to dex provider
			if res.Code == http.StatusSeeOther {
				location, err := result.Location()
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
				urlExchangeCodePhase := fmt.Sprintf("/api/v1/kubeconfig?code=%s&state=%s", test.AuthorizationCode, encodedState)

				// call kubeconfig endpoint after authentication
				// exchange code phase
				req = httptest.NewRequest("GET", urlExchangeCodePhase, strings.NewReader(""))
				res = httptest.NewRecorder()

				// create secure cookie
				if encoded, err := getSecureCookie().Encode(csrfCookieName, cookieValue); err == nil {
					// Drop a cookie on the recorder.
					http.SetCookie(res, &http.Cookie{Name: "csrf_token", Value: encoded})

					// Copy the Cookie over to a new Request
					req.Header.Add("Cookie", res.Header().Get("Set-Cookie"))
				}

				// act
				ep.ServeHTTP(res, req)
				defer res.Result().Body.Close()

				// validate
				assert.Equal(t, tc.ExpectedExchangeCodePhase.HTTPStatus, res.Code)
			}
		})
	}
}

func genTestKubeconfigKubermaticObjects() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		test.GenTestSeed(),
		// add some project
		test.GenDefaultProject(),
		// add a user
		test.GenDefaultUser(),
		test.GenAdminUser("john", "john@acme.com", true),
		// make the user the owner of the first project and the editor of the second
		test.GenDefaultOwnerBinding(),
		// add a cluster
		test.GenCluster(test.ClusterID, testClusterName, test.GenDefaultProject().Name, test.DefaultCreationTimestamp()),
	}
}

func marshalEncodeState(oidcState handlercommon.OIDCState) (string, error) {
	rawState, err := json.Marshal(oidcState)
	if err != nil {
		return "", err
	}
	encodedState := base64.StdEncoding.EncodeToString(rawState)
	return url.QueryEscape(encodedState), nil
}

func unmarshalState(rawState []byte) (handlercommon.OIDCState, error) {
	oidcState := handlercommon.OIDCState{}
	if err := json.Unmarshal(rawState, &oidcState); err != nil {
		return handlercommon.OIDCState{}, err
	}
	return oidcState, nil
}

func getSecureCookie() *securecookie.SecureCookie {
	return securecookie.New([]byte(""), nil)
}
