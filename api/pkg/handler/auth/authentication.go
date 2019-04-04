package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// contextKey defines a dedicated type for keys to use on contexts
type contextKey string

// OIDCExtractorVerifier is responsible for extracting auth data from a request
type OIDCExtractorVerifier interface {
	OIDCVerifier
	Extract(rq *http.Request) string
}

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
	// Users should use OIDCVerifier.Verify method to verify and extract claim from the token
	IDToken string
}

// OIDCIssuerVerifier combines OIDCIssuer and OIDCVerifier
type OIDCIssuerVerifier interface {
	OIDCIssuer
	OIDCVerifier
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

// OIDCVerifier knows how to verify OIDC token
type OIDCVerifier interface {
	// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
	// any additional checks depending on the Config, and returns the payload as OIDCClaims.
	Verify(ctx context.Context, token string) (OIDCClaims, error)
}

// OIDCClaims holds various claims extracted from the id_token
type OIDCClaims struct {
	Name    string
	Email   string
	Subject string
	Groups  []string
}

// TokenExtractor is an interface token extraction
type TokenExtractor interface {
	Extract(r *http.Request) string
}

// OpenIDClient implements OIDCIssuerVerifier and OIDCExtractorVerifier
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

// NewOpenIDClient returns an authentication middleware which authenticates against an openID server
func NewOpenIDClient(issuer, clientID, clientSecret, redirectURI string, extractor TokenExtractor, insecureSkipVerify bool) (*OpenIDClient, error) {
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
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecureSkipVerify},
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
func (o *OpenIDClient) Extract(rq *http.Request) string {
	return o.tokenExtractor.Extract(rq)
}

// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
// any additional checks depending on the Config, and returns the payload as OIDCClaims.
func (o *OpenIDClient) Verify(ctx context.Context, token string) (OIDCClaims, error) {
	if token == "" {
		return OIDCClaims{}, errors.New("token cannot be empty")
	}

	idToken, err := o.verifier.Verify(ctx, token)
	if err != nil {
		fmt.Printf("%v", err)
		return OIDCClaims{}, err
	}

	claims := map[string]interface{}{}
	err = idToken.Claims(&claims)
	if err != nil {
		return OIDCClaims{}, err
	}

	oidcClaims := OIDCClaims{}
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
func (e headerBearerTokenExtractor) Extract(r *http.Request) string {
	header := r.Header.Get(e.name)
	if len(header) < 7 {
		return ""
	}
	//strip BEARER/bearer/Bearer prefix
	return header[7:]
}

// NewQueryParamBearerTokenExtractor returns a token extractor which extracts the token from the given query parameter
func NewQueryParamBearerTokenExtractor(header string) TokenExtractor {
	return queryParamBearerTokenExtractor{name: header}
}

type queryParamBearerTokenExtractor struct {
	name string
}

// Extract extracts the bearer token from the query parameter
func (e queryParamBearerTokenExtractor) Extract(r *http.Request) string {
	return r.URL.Query().Get(e.name)
}

// NewCombinedExtractor returns an token extractor which tries a list of token extractors until it finds a token
func NewCombinedExtractor(extractors ...TokenExtractor) TokenExtractor {
	return combinedExtractor{extractors: extractors}
}

type combinedExtractor struct {
	extractors []TokenExtractor
}

// Extract extracts the token via the given token extractors. Returns as soon as it finds a token
func (c combinedExtractor) Extract(r *http.Request) string {
	for _, e := range c.extractors {
		t := e.Extract(r)
		if t != "" {
			return t
		}
	}
	return ""
}
