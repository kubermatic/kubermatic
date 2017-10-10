package handler

import (
	"context"
	"net/http"
	"strings"

	oidc "github.com/coreos/go-oidc"
	transporthttp "github.com/go-kit/kit/transport/http"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type contextKey int

const (
	// UserContextKey is the context key to retrieve the user object
	UserContextKey contextKey = 0
	// UserRoleKey is the role key for the default role "user"
	UserRoleKey = "user"
	// AdminRoleKey is the role key for the admin role
	AdminRoleKey = "admin"
)

// Authenticator is an interface for configurable authentication middlewares
type Authenticator interface {
	IsAuthenticated(transporthttp.DecodeRequestFunc) transporthttp.DecodeRequestFunc
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
func NewOpenIDAuthenticator(issuer, clientID string, extractor TokenExtractor) Authenticator {
	// Sanity check for config!
	_, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		glog.Fatal(err)
	}

	return openIDAuthenticator{
		issuer:         issuer,
		tokenExtractor: extractor,
		clientID:       clientID,
	}
}

// IsAuthenticated is a http middleware which checks against an openid server
func (o openIDAuthenticator) IsAuthenticated(h transporthttp.DecodeRequestFunc) transporthttp.DecodeRequestFunc {
	return transporthttp.DecodeRequestFunc(func(ctx context.Context, r *http.Request) (request interface{}, err error) {
		p, err := oidc.NewProvider(r.Context(), o.issuer)
		if err != nil {
			glog.Error(err)
			return nil, errors.NewNotAuthorized()
		}
		idTokenVerifier := p.Verifier(&oidc.Config{ClientID: o.clientID})
		token := o.tokenExtractor.Extract(r)
		glog.V(6).Infof("Extracted oauth token: %s", token)

		idToken, err := idTokenVerifier.Verify(r.Context(), token)
		if err != nil {
			glog.Error(err)
			return nil, errors.NewNotAuthorized()
		}

		claims := map[string]interface{}{}
		err = idToken.Claims(&claims)
		if err != nil {
			glog.Error(err)
			return nil, errors.NewNotAuthorized()
		}

		user := provider.User{
			Name:  claims["sub"].(string),
			Roles: map[string]struct{}{},
		}

		if user.Name == "" {
			glog.Error(err)
			return nil, errors.NewNotAuthorized()
		}

		roles := []string{UserRoleKey}
		md, ok := claims["app_metadata"].(map[string]interface{})
		if ok && md != nil {
			metaRoles, ok := md["roles"].([]interface{})
			if ok && metaRoles != nil {
				for _, r := range metaRoles {
					s, ok := r.(string)
					if ok && s != "" && s != UserRoleKey {
						roles = append(roles, s)
					}
				}
			}
		}

		for _, r := range roles {
			user.Roles[r] = struct{}{}
		}

		glog.V(6).Infof("Authenticated user: %s (Roles: %s)", user.Name, strings.Join(roles, ","))
		return h(context.WithValue(ctx, UserContextKey, user), r)
	})
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
	user interface{}
}

// NewTestAuthenticator returns an testing authentication middleware
func NewTestAuthenticator(user interface{}) Authenticator {
	return testAuthenticator{user: user}
}

// IsAuthenticated is a http middleware which checks against an openid server
func (o testAuthenticator) IsAuthenticated(h transporthttp.DecodeRequestFunc) transporthttp.DecodeRequestFunc {
	return transporthttp.DecodeRequestFunc(func(ctx context.Context, r *http.Request) (request interface{}, err error) {
		return h(ctx, r.WithContext(context.WithValue(r.Context(), UserContextKey, o.user)))
	})
}
