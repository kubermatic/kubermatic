package test

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"golang.org/x/oauth2"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/auth"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

const (
	// AuthorizationCode represents a shared secret used by IssuerVerifier
	// TODO: consider injecting it into IssuerVerifier
	AuthorizationCode = "fakeCode"
	// IDToken represents a shared fake token
	IDToken      = "fakeTokenId"
	refreshToken = "fakeRefreshToken"
	tokenURL     = "url:tokenURL"

	// IssuerURL holds test issuer URL
	IssuerURL = "url://dex"
	// IssuerClientID holds test issuer client ID
	IssuerClientID = "kubermatic"
	// IssuerClientSecret holds test issuer client secret
	IssuerClientSecret = "secret"
	issuerRedirectURL  = "/api/v1/kubeconfig"
)

var _ auth.OIDCIssuerVerifier = &IssuerVerifier{}

// testAuthenticator is a test stub that mocks apiv1.User
type testAuthenticator struct {
	user apiv1.User
}

// OicdProvider is a test stub that mocks *oidc.Provider
type OicdProvider struct {
	authURL  string
	tokenURL string
}

// NewAuthenticator returns an testing authentication middleware
func NewAuthenticator(user apiv1.User) auth.OIDCAuthenticator {
	return &testAuthenticator{user: user}
}

// Verifier is a convenient middleware that extracts the ID Token from the request,
// verifies it's been signed by the provider and creates apiv1.User from it
func (o *testAuthenticator) Verifier() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			_, ok := ctx.Value(middleware.AuthenticatedUserContextKey).(apiv1.User)
			if !ok {
				return nil, k8cerrors.NewNotAuthorized()
			}
			return next(ctx, request)
		}
	}
}

// Extractor knows how to extract the ID token from the request
func (o *testAuthenticator) Extractor() transporthttp.RequestFunc {
	return func(ctx context.Context, _ *http.Request) context.Context {
		return context.WithValue(ctx, middleware.AuthenticatedUserContextKey, o.user)
	}
}

// NewIssuerVerifier returns fake OIDC issuer and verifier
func NewIssuerVerifier() *IssuerVerifier {
	return &IssuerVerifier{
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
	issuer       string
	clientID     string
	clientSecret string
	redirectURI  string
	provider     *OicdProvider
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

// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
// any additional checks depending on the Config, and returns the payload as OIDCClaims.
func (o *IssuerVerifier) Verify(ctx context.Context, token string) (auth.OIDCClaims, error) {
	if o == nil {
		return auth.OIDCClaims{}, nil

	}
	if ctx == nil {
		return auth.OIDCClaims{}, nil
	}
	if token != IDToken {
		return auth.OIDCClaims{}, errors.New("incorrect code")
	}
	userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
	return auth.OIDCClaims{
		Email:  userInfo.Email,
		Groups: []string{userInfo.Group},
	}, nil
}
