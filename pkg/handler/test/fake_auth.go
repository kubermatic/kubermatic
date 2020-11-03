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

package test

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/auth"
)

const (
	// AuthorizationCode represents a shared secret used by IssuerVerifier
	// TODO: consider injecting it into IssuerVerifier
	AuthorizationCode = "fakeCode"
	// IDToken represents a shared fake token
	IDToken       = "fakeTokenId"
	IDViewerToken = "fakeViewerTokenId"
	refreshToken  = "fakeRefreshToken"
	tokenURL      = "url:tokenURL"

	// IssuerURL holds test issuer URL
	IssuerURL = "url://dex"
	// IssuerClientID holds test issuer client ID
	IssuerClientID = "kubermatic"
	// IssuerClientSecret holds test issuer client secret
	IssuerClientSecret = "secret"
	issuerRedirectURL  = "/api/v1/kubeconfig"
)

var _ auth.OIDCIssuerVerifier = &IssuerVerifier{}
var _ auth.TokenExtractorVerifier = &IssuerVerifier{}

// OicdProvider is a test stub that mocks *oidc.Provider
type OicdProvider struct {
	authURL  string
	tokenURL string
}

// NewFakeOIDCClient returns fake OIDC issuer and verifier
func NewFakeOIDCClient(user apiv1.User) *IssuerVerifier {
	return &IssuerVerifier{
		user:         user,
		issuer:       IssuerURL,
		clientID:     IssuerClientID,
		clientSecret: IssuerClientSecret,
		redirectURI:  issuerRedirectURL,
		provider: &OicdProvider{
			authURL:  IssuerURL,
			tokenURL: tokenURL,
		},
	}
}

// Endpoint returns the OAuth2 auth and token endpoints for the given provider.
func (p *OicdProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{AuthURL: p.authURL, TokenURL: p.tokenURL}
}

// IssuerVerifier is a test stub that mocks OIDC responses
type IssuerVerifier struct {
	user         apiv1.User
	issuer       string
	clientID     string
	clientSecret string
	redirectURI  string
	provider     *OicdProvider
}

// Extractor knows how to extract the ID token from the request
func (o *IssuerVerifier) Extract(_ *http.Request) (string, error) {
	return IDToken, nil
}

// AuthCodeURL returns a URL to OpenID provider's consent page
func (o *IssuerVerifier) AuthCodeURL(state string, offlineAsScope bool, scopes ...string) string {
	oauth2Config := o.oauth2Config(scopes...)
	options := oauth2.AccessTypeOnline
	if !offlineAsScope {
		options = oauth2.AccessTypeOffline
	}
	return oauth2Config.AuthCodeURL(state, options)
}

// oauth2Config return a oauth2Config
func (o *IssuerVerifier) oauth2Config(scopes ...string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  o.redirectURI,
	}
}

// Exchange converts an authorization code into a token.
func (o *IssuerVerifier) Exchange(ctx context.Context, code string) (auth.OIDCToken, error) {
	if code != AuthorizationCode {
		return auth.OIDCToken{}, errors.New("incorrect code")
	}

	return auth.OIDCToken{
		IDToken:      IDToken,
		RefreshToken: refreshToken,
	}, nil
}

// Verify parses a raw ID Token, verifies it's been signed by the provider, performs
// any additional checks depending on the Config, and returns the payload as TokenClaims.
func (o *IssuerVerifier) Verify(ctx context.Context, token string) (auth.TokenClaims, error) {
	if o == nil {
		return auth.TokenClaims{}, nil

	}
	if ctx == nil {
		return auth.TokenClaims{}, nil
	}
	if token != IDToken {
		return auth.TokenClaims{}, errors.New("incorrect code")
	}
	return auth.TokenClaims{
		Email:   o.user.Email,
		Subject: o.user.Email,
		Name:    o.user.Name,
		Groups:  []string{},
	}, nil
}
