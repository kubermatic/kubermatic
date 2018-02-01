package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/go-kit/kit/endpoint"
	transporthttp "github.com/go-kit/kit/transport/http"
	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

type contextKey int

const (
	// TokenUserContextKey is the context key to retrieve the user object
	TokenUserContextKey contextKey = 0
	// UserRoleKey is the role key for the default role "user"
	UserRoleKey = "user"
	// AdminRoleKey is the role key for the admin role
	AdminRoleKey = "kubermatic:admin"
)

// Authenticator  is responsible for extracting and verifying
// data to authenticate
type Authenticator interface {
	Verifier() endpoint.Middleware
	Extractor() transporthttp.RequestFunc
}

// TokenExtractor is an interface token extraction
type TokenExtractor interface {
	Extract(r *http.Request) string
}

type openIDAuthenticator struct {
	issuer         string
	tokenExtractor TokenExtractor
	clientID       string
}

// NewOpenIDAuthenticator returns an authentication middleware which authenticates against an openID server
func NewOpenIDAuthenticator(issuer, clientID string, extractor TokenExtractor) (Authenticator, error) {
	// Sanity check for config!
	_, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}

	return openIDAuthenticator{
		issuer:         issuer,
		tokenExtractor: extractor,
		clientID:       clientID,
	}, nil
}

func (o openIDAuthenticator) Verifier() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// This should only be called once!
			p, err := oidc.NewProvider(ctx, o.issuer)
			if err != nil {
				glog.Error(err)
				return nil, errors.NewNotAuthorized()
			}
			idTokenVerifier := p.Verifier(&oidc.Config{ClientID: o.clientID})
			t := ctx.Value(userRawToken)
			token, ok := t.(string)
			if !ok || token == "" {
				return nil, errors.NewNotAuthorized()
			}

			idToken, err := idTokenVerifier.Verify(ctx, token)
			if err != nil {
				glog.Error(err)
				return nil, errors.NewNotAuthorized()
			}

			// Verified
			claims := map[string]interface{}{}
			err = idToken.Claims(&claims)
			if err != nil {
				glog.Error(err)
				return nil, errors.NewNotAuthorized()
			}

			user := apiv1.User{
				ID:    claims["sub"].(string),
				Name:  claims["name"].(string),
				Email: claims["email"].(string),
				Roles: map[string]struct{}{},
			}

			if user.ID == "" {
				glog.Error(err)
				return nil, errors.NewNotAuthorized()
			}

			roles := []string{UserRoleKey}
			claimGroups, ok := claims["groups"].([]interface{})
			if ok && claimGroups != nil {
				for _, g := range claimGroups {
					s, ok := g.(string)
					if ok && s != "" && s != UserRoleKey {
						roles = append(roles, s)
					}
				}
			}

			for _, r := range roles {
				user.Roles[r] = struct{}{}
			}

			glog.V(6).Infof("Authenticated user: %s (Roles: %s)", user.ID, strings.Join(roles, ","))
			return next(context.WithValue(ctx, TokenUserContextKey, user), request)
		}
	}
}

type userToken int

const userRawToken userToken = 0

func (o openIDAuthenticator) Extractor() transporthttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		token := o.tokenExtractor.Extract(r)
		glog.V(6).Infof("Extracted oauth token: %s", token)
		return context.WithValue(ctx, userRawToken, token)
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

type testAuthenticator struct {
	user apiv1.User
}

// NewFakeAuthenticator returns an testing authentication middleware
func NewFakeAuthenticator(user apiv1.User) Authenticator {
	return testAuthenticator{user: user}
}

func (o testAuthenticator) Verifier() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			t := ctx.Value(userRawToken)
			token, ok := t.(apiv1.User)
			if !ok {
				return nil, errors.NewNotAuthorized()
			}
			return next(context.WithValue(ctx, TokenUserContextKey, token), request)
		}
	}
}

func (o testAuthenticator) Extractor() transporthttp.RequestFunc {
	return func(ctx context.Context, _ *http.Request) context.Context {
		return context.WithValue(ctx, userRawToken, o.user)
	}
}
