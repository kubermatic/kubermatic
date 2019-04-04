package auth

import (
	"context"
	"errors"
	"net/http"
)

// TokenClaims holds various claims extracted from the id_token
type TokenClaims struct {
	Name    string
	Email   string
	Subject string
	Groups  []string
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

// Verify calls all registered plugins to check the given token.
// This method stops when a token has been validated and doesn't try remaining plugins.
// If all plugins were checked an error is returned.
func (p *TokenVerifierPlugins) Verify(ctx context.Context, token string) (TokenClaims, error) {
	if len(p.plugins) == 0 {
		return TokenClaims{}, errors.New("cannot validate the token - no plugins registered")
	}
	for _, plugin := range p.plugins {
		if claims, err := plugin.Verify(ctx, token); err == nil {
			return claims, err
		}
	}
	return TokenClaims{}, errors.New("unable to verify the given token")
}

var _ TokenExtractor = &TokenExtractorPlugins{}

// TokenExtractorPlugins implements TokenExtractor
// by calling registered plugins for a token extraction
type TokenExtractorPlugins struct {
	plugins []TokenExtractor
}

// Extract calls all registered plugins to get a token from the given request.
// This method stops when a token has been found and doesn't try remaining plugins.
// If all plugins were checked an error is returned.
func (p *TokenExtractorPlugins) Extract(r *http.Request) (string, error) {
	if len(p.plugins) == 0 {
		return "", errors.New("cannot validate the token - no plugins registered")
	}
	for _, plugin := range p.plugins {
		if token, err := plugin.Extract(r); err == nil {
			return token, err
		}
	}
	return "", errors.New("unable to verify the given token")
}
