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
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

// TokenClaims holds various claims extracted from the id_token.
type TokenClaims struct {
	Name    string
	Email   string
	Subject string
	Groups  []string
	Expiry  apiv1.Time
}

// TokenExtractorVerifier combines TokenVerifier and TokenExtractor interfaces.
type TokenExtractorVerifier interface {
	TokenVerifier
	TokenExtractor
}

// TokenExtractor is an interface that knows how to extract a token.
type TokenExtractor interface {
	// Extract gets a token from the given HTTP request
	Extract(r *http.Request) (string, error)
}

// TokenVerifier knows how to verify a token.
type TokenVerifier interface {
	// Verify parses a raw ID Token, verifies it's been signed by the provider, performs
	// any additional checks depending on the Config, and returns the payload as TokenClaims.
	Verify(ctx context.Context, token string) (TokenClaims, error)
}
