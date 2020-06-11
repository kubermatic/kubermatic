package auth

import (
	"context"
	"errors"
	"net/http"

	k8cerrors "github.com/kubermatic/kubermatic/pkg/util/errors"
)

// TokenClaims holds various claims extracted from the id_token
type TokenClaims struct {
	Name    string
	Email   string
	Subject string
	Groups  []string
}

// TokenExtractorVerifier combines TokenVerifier and TokenExtractor interfaces
type TokenExtractorVerifier interface {
	TokenVerifier
	TokenExtractor
}

// TokenExtractor is an interface that knows how to extract a token
type TokenExtractor interface {
	// Extract gets a token from the given HTTP request
	Extract(r *http.Request) (string, error)
}

// TokenVerifier knows how to verify a token
type TokenVerifier interface {
	// Verify parses a raw ID Token, verifies it's been signed by the provider, preforms
	// any additional checks depending on the Config, and returns the payload as TokenClaims.
	Verify(ctx context.Context, token string) (TokenClaims, error)
}

var _ TokenVerifier = &TokenVerifierPlugins{}

// TokenVerifierPlugins implements TokenVerifier interface
// by calling registered plugins for a token verification
type TokenVerifierPlugins struct {
	plugins []TokenVerifier
}

// NewTokenVerifierPlugins creates a new instance of TokenVerifierPlugins with the given plugins
func NewTokenVerifierPlugins(plugins []TokenVerifier) *TokenVerifierPlugins {
	return &TokenVerifierPlugins{plugins}
}

// Verify calls all registered plugins to check the given token.
// This method stops when a token has been validated and doesn't try remaining plugins.
// If all plugins were checked an error is returned.
func (p *TokenVerifierPlugins) Verify(ctx context.Context, token string) (TokenClaims, error) {
	if len(p.plugins) == 0 {
		return TokenClaims{}, errors.New("cannot validate the token - no plugins registered")
	}
	var errList []error
	for _, plugin := range p.plugins {
		claims, err := plugin.Verify(ctx, token)
		if err == nil {
			return claims, err
		}
		errList = append(errList, err)
	}
	return TokenClaims{}, k8cerrors.NewAggregate(errList)
}

var _ TokenExtractor = &TokenExtractorPlugins{}

// TokenExtractorPlugins implements TokenExtractor
// by calling registered plugins for a token extraction
type TokenExtractorPlugins struct {
	plugins []TokenExtractor
}

// NewTokenExtractorPlugins creates a new instance of TokenExtractorPlugins with the given plugins
func NewTokenExtractorPlugins(plugins []TokenExtractor) *TokenExtractorPlugins {
	return &TokenExtractorPlugins{plugins}
}

// Extract calls all registered plugins to get a token from the given request.
// This method stops when a token has been found and doesn't try remaining plugins.
// If all plugins were checked an error is returned.
func (p *TokenExtractorPlugins) Extract(r *http.Request) (string, error) {
	if len(p.plugins) == 0 {
		return "", errors.New("cannot validate the token - no plugins registered")
	}
	var errList []error
	for _, plugin := range p.plugins {
		token, err := plugin.Extract(r)
		if err == nil {
			return token, err
		}
		errList = append(errList, err)
	}
	return "", k8cerrors.NewAggregate(errList)
}
