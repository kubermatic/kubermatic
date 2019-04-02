package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// ServiceAccountAuthClient implements TokenExtractorVerifier interface
type ServiceAccountAuthClient struct {
	headerBearerTokenExtractor TokenExtractor
	jwtTokenAuthenticator      serviceaccount.TokenAuthenticator
	saTokenProvider            provider.PrivilegedServiceAccountTokenProvider
}

var _ TokenExtractorVerifier = &ServiceAccountAuthClient{}

// NewServiceAccountAuthClient returns a client that knows how to read and verify service account's tokens
func NewServiceAccountAuthClient(headerBearerTokenExtractor TokenExtractor, jwtTokenAuthenticator serviceaccount.TokenAuthenticator, saTokenProvider provider.PrivilegedServiceAccountTokenProvider) *ServiceAccountAuthClient {
	return &ServiceAccountAuthClient{headerBearerTokenExtractor: headerBearerTokenExtractor, jwtTokenAuthenticator: jwtTokenAuthenticator, saTokenProvider: saTokenProvider}
}

// Extractor knows how to extract the ID token from the request
func (s *ServiceAccountAuthClient) Extract(rq *http.Request) (string, error) {
	return s.headerBearerTokenExtractor.Extract(rq)
}

// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
// any additional checks depending on the Config, and returns the payload as TokenClaims.
func (s *ServiceAccountAuthClient) Verify(ctx context.Context, token string) (TokenClaims, error) {
	_, customClaims, err := s.jwtTokenAuthenticator.Authenticate(token)
	if err != nil {
		return TokenClaims{}, err
	}

	tokenList, err := s.saTokenProvider.ListUnsecured(&provider.ServiceAccountTokenListOptions{TokenName: customClaims.TokenID})
	if kerrors.IsNotFound(err) {
		return TokenClaims{}, fmt.Errorf("sa: the token %s has been revoked for %s", customClaims.TokenID, customClaims.Email)
	}
	if len(tokenList) > 1 {
		return TokenClaims{}, fmt.Errorf("sa: found more than one token with the given id %s", customClaims.TokenID)
	}

	return TokenClaims{
		Name:    customClaims.TokenID,
		Email:   customClaims.Email,
		Subject: customClaims.Email,
	}, nil
}
