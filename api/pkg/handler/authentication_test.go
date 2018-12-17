package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"golang.org/x/oauth2"
)

const (
	testAuthorizationCode = "fakeCode"
	testIDToken           = "fakeTokenId"
	testRefreshToken      = "fakeRefreshToken"
	testTokenURL          = "url:tokenURL"

	testIssuerURL          = "url://dex"
	testIssuerRedirectURL  = "/api/v1/kubeconfig"
	testIssuerClientID     = "kubermatic"
	testIssuerClientSecret = "secret"

	testClusterID = "AbcClusterID"
)

var _ OIDCIssuerVerifier = FakeIssuerVerifier{}

// testAuthenticator is a test stub that mocks apiv1.User
type testAuthenticator struct {
	user apiv1.User
}

// FakeOicdProvider is a test stub that mocks *oidc.Provider
type FakeOicdProvider struct {
	authURL  string
	tokenURL string
}

// NewFakeAuthenticator returns an testing authentication middleware
func NewFakeAuthenticator(user apiv1.User) OIDCAuthenticator {
	return testAuthenticator{user: user}
}

// Verifier is a convenient middleware that extracts the ID Token from the request,
// verifies it's been signed by the provider and creates apiv1.User from it
func (o testAuthenticator) Verifier() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			_, ok := ctx.Value(authenticatedUserContextKey).(apiv1.User)
			if !ok {
				return nil, k8cerrors.NewNotAuthorized()
			}
			return next(ctx, request)
		}
	}
}

// Extractor knows how to extract the ID token from the request
func (o testAuthenticator) Extractor() transporthttp.RequestFunc {
	return func(ctx context.Context, _ *http.Request) context.Context {
		return context.WithValue(ctx, authenticatedUserContextKey, o.user)
	}
}

// NewFakeIssuerVerifier returns fake OIDC issuer and verifier
func NewFakeIssuerVerifier() FakeIssuerVerifier {
	return FakeIssuerVerifier{
		issuer:       testIssuerURL,
		clientID:     testIssuerClientID,
		clientSecret: testIssuerClientSecret,
		redirectURI:  testIssuerRedirectURL,
		provider: &FakeOicdProvider{
			authURL:  testIssuerURL,
			tokenURL: testTokenURL,
		},
	}
}

// Endpoint returns the OAuth2 auth and token endpoints for the given provider.
func (p *FakeOicdProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{AuthURL: p.authURL, TokenURL: p.tokenURL}
}

// FakeIssuerVerifier is a test stub that mocks OIDC responses
type FakeIssuerVerifier struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURI  string
	provider     *FakeOicdProvider
}

// AuthCodeURL returns a URL to OpenID provider's consent page
func (o FakeIssuerVerifier) AuthCodeURL(state string, offlineAsScope bool, scopes ...string) string {
	oauth2Config := o.oauth2Config(scopes...)
	options := oauth2.AccessTypeOnline
	if !offlineAsScope {
		options = oauth2.AccessTypeOffline
	}
	return oauth2Config.AuthCodeURL(state, options)
}

// oauth2Config return a oauth2Config
func (o *FakeIssuerVerifier) oauth2Config(scopes ...string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.clientID,
		ClientSecret: o.clientSecret,
		Endpoint:     o.provider.Endpoint(),
		Scopes:       scopes,
		RedirectURL:  o.redirectURI,
	}
}

// Exchange converts an authorization code into a token.
func (o FakeIssuerVerifier) Exchange(ctx context.Context, code string) (OIDCToken, error) {
	if code != testAuthorizationCode {
		return OIDCToken{}, errors.New("incorrect code")
	}

	return OIDCToken{
		IDToken:      testIDToken,
		RefreshToken: testRefreshToken,
	}, nil
}

// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
// any additional checks depending on the Config, and returns the payload as OIDCClaims.
func (o FakeIssuerVerifier) Verify(ctx context.Context, token string) (OIDCClaims, error) {
	if token != testIDToken {
		return OIDCClaims{}, errors.New("incorrect code")
	}
	userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
	return OIDCClaims{
		Email:  userInfo.Email,
		Groups: []string{userInfo.Group},
	}, nil
}
