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

package kubernetesdashboard

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	"k8c.io/kubermatic/v2/pkg/handler/auth"
	commonv2 "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	nonceCookieName   = "nonce"
	nonceCookieMaxAge = 180
)

type loginHandler struct {
	baseHandler

	oidcConfig         common.OIDCConfiguration
	oidcIssuerVerifier auth.OIDCIssuerVerifier
	settingsProvider   provider.SettingsProvider
	secureCookie       *securecookie.SecureCookie
}

func (this *loginHandler) Middlewares(middlewares ...endpoint.Middleware) Handler {
	this.middlewares = middlewares
	return this
}

func (this *loginHandler) Options(options ...httptransport.ServerOption) Handler {
	this.options = options
	return this
}

func (this *loginHandler) Install(router *mux.Router) {
	router.Methods(http.MethodGet).
		Path("/dashboard/login").
		Queries("projectID", "{projectID}", "clusterID", "{clusterID}").
		Handler(this.redirectHandler())

	router.Methods(http.MethodGet).
		Path("/dashboard/login").
		Queries("state", "{state}", "code", "{code}").
		Handler(this.oidcCallbackHandler())
}

func (this *loginHandler) decodeInitialRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return NewInitialRequest(r), nil
}

func (this *loginHandler) encodeInitialResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	loginResponse := response.(*LoginResponse)

	encodedNonceCookie, err := this.getEncodedNonceCookie(loginResponse.nonce, this.oidcConfig.CookieSecureMode, nonceCookieMaxAge)
	if err != nil {
		return err
	}

	http.SetCookie(w, encodedNonceCookie)
	http.Redirect(w, loginResponse.Request, loginResponse.authURL, http.StatusSeeOther)
	return nil
}

// swagger:route GET /api/v2/dashboard/login
//
//	Redirects to the OIDC page for additional user authentication.
//
//	Parameters:
//		+ name: projectID
//		  in: query
//		  required: true
//		  type: string
//		+ name: clusterID
//		  in: query
//		  required: true
//		  type: string
//
//	Responses:
//		default: empty
func (this *loginHandler) redirectHandler() http.Handler {
	return httptransport.NewServer(
		this.chain(this.redirect),
		this.decodeInitialRequest,
		this.encodeInitialResponse,
		this.options...,
	)
}

func (this *loginHandler) redirect(ctx context.Context, request interface{}) (response interface{}, err error) {
	loginRequest := request.(*InitialRequest)
	nonce := rand.String(rand.IntnRange(10, 15))
	scopes := []string{"openid", "email"}

	// Make sure the global settings have the Dashboard integration enabled.
	if err := isEnabled(ctx, this.settingsProvider); err != nil {
		return nil, err
	}

	if this.oidcConfig.OfflineAccessAsScope {
		scopes = append(scopes, "offline_access")
	}

	state, err := this.encodeOIDCState(nonce, loginRequest.ProjectID, loginRequest.ClusterID)
	if err != nil {
		return nil, err
	}

	// get the redirect uri
	redirectURI, err := this.oidcIssuerVerifier.GetRedirectURI(loginRequest.Request.URL.Path)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Request: loginRequest.Request,
		authURL: this.oidcIssuerVerifier.AuthCodeURL(state, this.oidcConfig.OfflineAccessAsScope, redirectURI, scopes...),
		nonce:   nonce,
	}, nil
}

func (this *loginHandler) getEncodedNonceCookie(nonce string, secureMode bool, maxAge int) (*http.Cookie, error) {
	encoded, err := this.secureCookie.Encode(nonceCookieName, nonce)
	if err != nil {
		return nil, fmt.Errorf("the encode cookie failed: %w", err)
	}

	return &http.Cookie{
		Name:     nonceCookieName,
		Value:    encoded,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secureMode,
		SameSite: http.SameSiteLaxMode,
	}, nil
}

func (this *loginHandler) decodeOIDCCallbackRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return NewOIDCCallbackRequest(r), nil
}

func (this *loginHandler) encodeOIDCCallbackResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	callbackResponse := response.(*OIDCCallbackResponse)

	cookie, err := this.getEncodedNonceCookie("", this.oidcConfig.CookieSecureMode, -1)
	if err != nil {
		return err
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, callbackResponse.Request, this.getProxyURI(callbackResponse.projectID, callbackResponse.clusterID, callbackResponse.token), http.StatusSeeOther)
	return nil
}

// swagger:route GET /api/v2/dashboard/login
//
//	OIDC redirectURI endpoint that gets called after successful user authentication. Checks the token and redirects
//	user to the dashboard proxy endpoint: proxyHandler.storeTokenHandler
//
//	Parameters:
//		+ name: state
//		  in: query
//		  required: true
//		  type: string
//		+ name: code
//		  in: query
//		  required: true
//		  type: string
//
//	Responses:
//		default: empty
func (this *loginHandler) oidcCallbackHandler() http.Handler {
	return httptransport.NewServer(
		this.chain(this.oidcCallback),
		this.decodeOIDCCallbackRequest,
		this.encodeOIDCCallbackResponse,
		this.options...,
	)
}

func (this *loginHandler) oidcCallback(ctx context.Context, request interface{}) (response interface{}, err error) {
	oidcCallbackRequest := request.(*OIDCCallbackRequest)

	state, err := this.decodeOIDCState(oidcCallbackRequest.State)
	if err != nil {
		return nil, err
	}

	nonce, err := this.getDecodedNonce(oidcCallbackRequest.Request)
	if err != nil {
		return nil, err
	}

	if state.Nonce != nonce {
		return nil, utilerrors.NewBadRequest("incorrect value of state parameter: %s", state.Nonce)
	}

	// get the redirect uri
	redirectURI, err := this.oidcIssuerVerifier.GetRedirectURI(oidcCallbackRequest.Request.URL.Path)
	if err != nil {
		return nil, err
	}

	token, err := this.exchange(ctx, oidcCallbackRequest.Code, redirectURI)
	if err != nil {
		return nil, err
	}

	return &OIDCCallbackResponse{
		Request:   oidcCallbackRequest.Request,
		projectID: state.ProjectID,
		clusterID: state.ClusterID,
		token:     token,
	}, nil
}

func (this *loginHandler) exchange(ctx context.Context, code, overwriteRedirectURI string) (string, error) {
	oidcTokens, err := this.oidcIssuerVerifier.Exchange(ctx, code, overwriteRedirectURI)
	if err != nil {
		return "", utilerrors.NewBadRequest("error while exchanging oidc code for token: %v", err)
	}

	if len(oidcTokens.RefreshToken) == 0 {
		return "", utilerrors.NewBadRequest("the refresh token is missing but required, try setting/unsetting \"oidc-offline-access-as-scope\" command line flag")
	}

	claims, err := this.oidcIssuerVerifier.Verify(ctx, oidcTokens.IDToken)
	if err != nil {
		return "", utilerrors.New(http.StatusUnauthorized, err.Error())
	}

	if len(claims.Email) == 0 {
		return "", utilerrors.NewBadRequest("the token doesn't contain the mandatory \"email\" claim")
	}

	return oidcTokens.IDToken, nil
}

func (this *loginHandler) getDecodedNonce(r *http.Request) (nonce string, err error) {
	cookie, err := r.Cookie(nonceCookieName)
	if err != nil {
		return
	}

	err = this.secureCookie.Decode(nonceCookieName, cookie.Value, &nonce)
	return
}

func (this *loginHandler) getProxyURI(projectID string, clusterID string, token string) string {
	return fmt.Sprintf("/api/v2/projects/%s/clusters/%s/dashboard/proxy?token=%s", projectID, clusterID, token)
}

func (this *loginHandler) encodeOIDCState(nonce string, projectID string, clusterID string) (string, error) {
	oidcState := commonv2.OIDCState{
		Nonce:     nonce,
		ClusterID: clusterID,
		ProjectID: projectID,
	}

	rawState, err := json.Marshal(oidcState)
	if err != nil {
		return "", err
	}

	encodedState := base64.StdEncoding.EncodeToString(rawState)
	urlSafeState := url.QueryEscape(encodedState)

	return urlSafeState, nil
}

func (this *loginHandler) decodeOIDCState(state string) (*commonv2.OIDCState, error) {
	unescapedState, err := url.QueryUnescape(state)
	if err != nil {
		return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected url encoded value: %v", err)
	}
	rawState, err := base64.StdEncoding.DecodeString(unescapedState)
	if err != nil {
		return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected base64 encoded value: %v", err)
	}
	oidcState := commonv2.OIDCState{}
	if err = json.Unmarshal(rawState, &oidcState); err != nil {
		return nil, utilerrors.NewBadRequest("incorrect value of state parameter, expected json encoded value: %v", err)
	}

	return &oidcState, nil
}

func NewLoginHandler(oidcConfig common.OIDCConfiguration, oidcIssuerVerifier auth.OIDCIssuerVerifier, settingsProvider provider.SettingsProvider) Handler {
	return &loginHandler{
		oidcConfig:         oidcConfig,
		oidcIssuerVerifier: oidcIssuerVerifier,
		settingsProvider:   settingsProvider,
		secureCookie:       securecookie.New([]byte(oidcConfig.CookieHashKey), nil),
	}
}
