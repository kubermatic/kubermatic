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

package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

// OIDCToken represents the credentials used to authorize
// the requests to access protected resources on the OAuth 2.0
// provider's backend.
type OIDCToken struct {
	// AccessToken is the token that authorizes and authenticates
	// the requests.
	AccessToken string

	// RefreshToken is a token that's used by the application
	// (as opposed to the user) to refresh the access token
	// if it expires.
	RefreshToken string

	// Expiry is the optional expiration time of the access token.
	//
	// If zero, TokenSource implementations will reuse the same
	// token forever and RefreshToken or equivalent
	// mechanisms for that TokenSource will not be used.
	Expiry time.Time

	// IDToken is the token that contains claims about authenticated user
	//
	// Users should use TokenVerifier.Verify method to verify and extract claim from the token
	IDToken string
}

// OIDCIssuerVerifier combines OIDCIssuer and TokenVerifier
type OIDCIssuerVerifier interface {
	OIDCIssuer
	TokenVerifier
}

// OIDCIssuer exposes methods for getting OIDC tokens
type OIDCIssuer interface {
	// AuthCodeURL returns a URL to OpenID provider's consent page
	// that asks for permissions for the required scopes explicitly.
	//
	// state is a token to protect the user from CSRF attacks. You must
	// always provide a non-zero string and validate that it matches the
	// the state query parameter on your redirect callback.
	// See http://tools.ietf.org/html/rfc6749#section-10.12 for more info.
	AuthCodeURL(state string, offlineAsScope bool, scopes ...string) string

	// Exchange converts an authorization code into a token.
	Exchange(ctx context.Context, code string) (OIDCToken, error)
}

// OpenIDClient implements OIDCIssuerVerifier and TokenExtractorVerifier
type OpenIDClient struct {
	issuer         string
	tokenExtractor TokenExtractor
	clientID       string
	clientSecret   string
	redirectURI    string
	verifier       *oidc.IDTokenVerifier
	provider       *oidc.Provider
	httpClient     *http.Client
}

// NewOpenIDClient returns an authentication middleware which authenticates against an openID server.
// If rootCertificates is nil, the host's root CAs will be used.
func NewOpenIDClient(issuer, clientID, clientSecret, redirectURI string, extractor TokenExtractor, insecureSkipVerify bool, rootCertificates *x509.CertPool) (*OpenIDClient, error) {
	ctx := context.Background()
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs:            rootCertificates,
			InsecureSkipVerify: insecureSkipVerify,
		},
	}
	client := &http.Client{Transport: tr}

	p, err := oidc.NewProvider(context.WithValue(ctx, oauth2.HTTPClient, client), issuer)
	if err != nil {
		return nil, err
	}

	return &OpenIDClient{
		issuer:         issuer,
		tokenExtractor: extractor,
		clientID:       clientID,
		clientSecret:   clientSecret,
		redirectURI:    redirectURI,
		verifier:       p.Verifier(&oidc.Config{ClientID: clientID}),
		provider:       p,
		httpClient:     client,
	}, nil
}

// Extractor knows how to extract the ID token from the request
func (o *OpenIDClient) Extract(rq *http.Request) (string, error) {
	return o.tokenExtractor.Extract(rq)
}

// Verify parses a raw ID Token, verifies it's been signed by the provider, performs
// any additional checks depending on the Config, and returns the payload as TokenClaims.
func (o *OpenIDClient) Verify(ctx context.Context, token string) (TokenClaims, error) {
	if token == "" {
		return TokenClaims{}, errors.New("token cannot be empty")
	}

	idToken, err := o.verifier.Verify(ctx, token)
	if err != nil {
		return TokenClaims{}, err
	}

	claims := map[string]interface{}{}
	err = idToken.Claims(&claims)
	if err != nil {
		return TokenClaims{}, err
	}

	oidcClaims := TokenClaims{}
	if rawName, found := claims["name"]; found {
		oidcClaims.Name = rawName.(string)
	}
	if rawEmail, found := claims["email"]; found {
		oidcClaims.Email = rawEmail.(string)
	}
	if rawSub, found := claims["sub"]; found {
		oidcClaims.Subject = rawSub.(string)
	}
	if rawGroups, found := claims["groups"]; found {
		for _, rawGroup := range rawGroups.([]interface{}) {
			if group, ok := rawGroup.(string); ok {
				oidcClaims.Groups = append(oidcClaims.Groups, group)
			}
		}
	}
	if rawExp, found := claims["exp"]; found {
		exp := rawExp.(float64)
		secs := int64(exp)
		nsecs := int64((exp - float64(secs)) * 1e9)
		oidcClaims.Expiry = apiv1.NewTime(time.Unix(secs, nsecs))
	}

	return oidcClaims, nil
}

// AuthCodeURL returns a URL to OpenID provider's consent page
// that asks for permissions for the required scopes explicitly.
//
// State is a token to protect the user from CSRF attacks. You must
// always provide a non-zero string and validate that it matches the
// the state query parameter on your redirect callback.
// See http://tools.ietf.org/html/rfc6749#section-10.12 for more info.
func (o *OpenIDClient) AuthCodeURL(state string, offlineAsScope bool, scopes ...string) string {
	oauth2Config := o.oauth2Config(scopes...)
	options := oauth2.AccessTypeOnline
	if !offlineAsScope {
		options = oauth2.AccessTypeOffline
	}
	return oauth2Config.AuthCodeURL(state, options)
}

// Exchange converts an authorization code into a token.
func (o *OpenIDClient) Exchange(ctx context.Context, code string) (OIDCToken, error) {
	clientCtx := oidc.ClientContext(ctx, o.httpClient)
	oauth2Config := o.oauth2Config()

	tokens, err := oauth2Config.Exchange(clientCtx, code)
	if err != nil {
		return OIDCToken{}, err
	}

	oidcToken := OIDCToken{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, Expiry: tokens.Expiry}
	if rawIDToken, ok := tokens.Extra("id_token").(string); ok {
		oidcToken.IDToken = rawIDToken
	}

	return oidcToken, nil
}

func (o *OpenIDClient) oauth2Config(scopes ...string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  o.redirectURI,
	}
}

// NewHeaderBearerTokenExtractor returns a token extractor which extracts the token from the given header
func NewHeaderBearerTokenExtractor(header string) TokenExtractor {
	return headerBearerTokenExtractor{name: header}
}

type headerBearerTokenExtractor struct {
	name string
}

// Extract extracts the bearer token from the header
func (e headerBearerTokenExtractor) Extract(r *http.Request) (string, error) {
	header := r.Header.Get(e.name)
	if len(header) < 7 {
		return "", fmt.Errorf("haven't found a Bearer token in the %s header", e.name)
	}
	//strip BEARER/bearer/Bearer prefix
	return header[7:], nil
}

// NewQueryParamBearerTokenExtractor returns a token extractor which extracts the token from the given query parameter
func NewQueryParamBearerTokenExtractor(header string) TokenExtractor {
	return queryParamBearerTokenExtractor{name: header}
}

type queryParamBearerTokenExtractor struct {
	name string
}

// Extract extracts the bearer token from the query parameter
func (e queryParamBearerTokenExtractor) Extract(r *http.Request) (string, error) {
	val := r.URL.Query().Get(e.name)
	if len(val) == 0 {
		return "", fmt.Errorf("haven't found an OIDC token in the query %q param ", e.name)
	}
	return val, nil
}

func NewCookieHeaderBearerTokenExtractor(header string) TokenExtractor {
	return cookieHeaderBearerTokenExtractor{name: header}
}

type cookieHeaderBearerTokenExtractor struct {
	name string
}

func (e cookieHeaderBearerTokenExtractor) Extract(r *http.Request) (string, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", fmt.Errorf("haven't found a Bearer token in the Cookie header %s: %v", e.name, err)
	}

	return cookie.Value, nil
}

// NewCombinedExtractor returns an token extractor which tries a list of token extractors until it finds a token
func NewCombinedExtractor(extractors ...TokenExtractor) TokenExtractor {
	return combinedExtractor{extractors: extractors}
}

type combinedExtractor struct {
	extractors []TokenExtractor
}

// Extract extracts the token via the given token extractors. Returns as soon as it finds a token
func (c combinedExtractor) Extract(r *http.Request) (string, error) {
	for _, extractor := range c.extractors {
		token, err := extractor.Extract(r)
		if err == nil {
			return token, nil
		}
	}
	return "", fmt.Errorf("haven't found an OIDC token, tried %d extractors", len(c.extractors))
}
