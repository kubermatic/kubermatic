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
	"fmt"
	"net/http"

	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/serviceaccount"

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

// Verify parses a raw ID Token, verifies it's been signed by the provider, performs
// any additional checks depending on the Config, and returns the payload as TokenClaims.
func (s *ServiceAccountAuthClient) Verify(ctx context.Context, token string) (TokenClaims, error) {
	_, customClaims, err := s.jwtTokenAuthenticator.Authenticate(token)
	if err != nil {
		return TokenClaims{}, err
	}

	tokenList, err := s.saTokenProvider.ListUnsecured(&provider.ServiceAccountTokenListOptions{TokenID: customClaims.TokenID})
	if kerrors.IsNotFound(err) {
		return TokenClaims{}, fmt.Errorf("sa: the token %s has been revoked for %s", customClaims.TokenID, customClaims.Email)
	}
	if len(tokenList) > 1 {
		return TokenClaims{}, fmt.Errorf("sa: found more than one token with the given id %s", customClaims.TokenID)
	}
	rawToken := tokenList[0]
	tokenFromDB, ok := rawToken.Data["token"]
	if !ok {
		return TokenClaims{}, fmt.Errorf("sa: cannot verify the token (%s) because the corresponding token in the database is invalid", customClaims.TokenID)
	}
	if string(tokenFromDB) != token {
		return TokenClaims{}, fmt.Errorf("sa: the token %s has been revoked for %s", customClaims.TokenID, customClaims.Email)
	}

	return TokenClaims{
		Name:    customClaims.TokenID,
		Email:   customClaims.Email,
		Subject: customClaims.Email,
	}, nil
}
