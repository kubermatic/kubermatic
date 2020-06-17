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

package serviceaccount

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Now stubbed out to allow testing
var Now = time.Now

// TokenGenerator declares the method to generate JWT token
type TokenGenerator interface {
	// Generate generates a token which will identify the given
	// ServiceAccount. privateClaims is an interface that will be
	// serialized into the JWT payload JSON encoding at the root level of
	// the payload object. Public claims take precedent over private
	// claims i.e. if both claims and privateClaims have an "exp" field,
	// the value in claims will be used.
	Generate(claims *jwt.Claims, customClaims *CustomTokenClaim) (string, error)
}

// TokenAuthenticator declares the method to check JWT token
type TokenAuthenticator interface {
	// Authenticate checks given token and transform it to custom claim object
	Authenticate(tokenData string) (*jwt.Claims, *CustomTokenClaim, error)
}

// CustomTokenClaim represents authenticated user
type CustomTokenClaim struct {
	Email     string `json:"email,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	TokenID   string `json:"token_id,omitempty"`
}

func Claims(email, projectID, tokenID string) (*jwt.Claims, *CustomTokenClaim) {

	sc := &jwt.Claims{
		IssuedAt:  jwt.NewNumericDate(Now()),
		NotBefore: jwt.NewNumericDate(Now()),
		Expiry:    jwt.NewNumericDate(Now().AddDate(3, 0, 0)),
	}
	pc := &CustomTokenClaim{
		Email:     email,
		ProjectID: projectID,
		TokenID:   tokenID,
	}

	return sc, pc
}

// JWTTokenGenerator returns a TokenGenerator that generates signed JWT tokens, using the given privateKey.
func JWTTokenGenerator(privateKey []byte) (TokenGenerator, error) {
	if err := ValidateKey(privateKey); err != nil {
		return nil, err
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: privateKey}, &jose.SignerOptions{})
	if err != nil {
		return nil, err
	}
	return &jwtTokenGenerator{
		signer: signer,
	}, nil
}

type jwtTokenGenerator struct {
	signer jose.Signer
}

type jwtTokenAuthenticator struct {
	key interface{}
}

// Generate generates new token from claims
func (j *jwtTokenGenerator) Generate(claims *jwt.Claims, customClaims *CustomTokenClaim) (string, error) {
	// claims are applied in reverse precedence
	return jwt.Signed(j.signer).
		Claims(customClaims).
		Claims(claims).
		CompactSerialize()
}

// JWTTokenAuthenticator authenticates tokens as JWT tokens produced by JWTTokenGenerator
func JWTTokenAuthenticator(privateKey []byte) TokenAuthenticator {
	return &jwtTokenAuthenticator{
		key: privateKey,
	}
}

// Authenticate decrypts signed token data to CustomTokenClaim object and checks if token expired
func (a *jwtTokenAuthenticator) Authenticate(tokenData string) (*jwt.Claims, *CustomTokenClaim, error) {

	tok, err := jwt.ParseSigned(tokenData)
	if err != nil {
		return nil, nil, err
	}

	public := &jwt.Claims{}
	customClaims := &CustomTokenClaim{}

	if err := tok.Claims(a.key, customClaims, public); err != nil {
		return nil, nil, err
	}

	err = public.Validate(jwt.Expected{
		Time: Now(),
	})
	switch {
	case err == nil:
	case err == jwt.ErrExpired:
		return nil, nil, fmt.Errorf("token has expired")
	default:
		return nil, nil, fmt.Errorf("token could not be validated due to error: %v", err)
	}

	return public, customClaims, nil
}

func ValidateKey(privateKey []byte) error {
	if len(privateKey) == 0 {
		return fmt.Errorf("the signing key can not be empty")
	}
	if len(privateKey) < 32 {
		return fmt.Errorf("the signing key is to short, use 32 bytes or longer")
	}
	return nil
}
