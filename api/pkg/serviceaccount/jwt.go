package serviceaccount

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Now stubbed out to allow testing
var Now = time.Now

type TokenGenerator interface {
	// Generate generates a token which will identify the given
	// ServiceAccount. privateClaims is an interface that will be
	// serialized into the JWT payload JSON encoding at the root level of
	// the payload object. Public claims take precedent over private
	// claims i.e. if both claims and privateClaims have an "exp" field,
	// the value in claims will be used.
	Generate(claims *jwt.Claims, privateClaims *TokenClaim) (string, error)
}

type TokenAuthenticator interface {
	// Authenticate checks given token and transform it to custom claim object
	Authenticate(tokenData string) (*jwt.Claims, *TokenClaim, error)
}

type TokenClaim struct {
	Email     string `json:"email,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	TokenID   string `json:"token_id,omitempty"`
}

func Claims(email, projectID, tokenID string) (*jwt.Claims, *TokenClaim) {

	sc := &jwt.Claims{
		IssuedAt:  jwt.NewNumericDate(Now()),
		NotBefore: jwt.NewNumericDate(Now()),
		Expiry:    jwt.NewNumericDate(Now().AddDate(3, 0, 0)),
	}
	pc := &TokenClaim{
		Email:     email,
		ProjectID: projectID,
		TokenID:   tokenID,
	}

	return sc, pc
}

// JWTTokenGenerator returns a TokenGenerator that generates signed JWT tokens, using the given privateKey.
func JWTTokenGenerator(privateKey []byte) (TokenGenerator, error) {
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
func (j *jwtTokenGenerator) Generate(claims *jwt.Claims, customClaims *TokenClaim) (string, error) {
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

// Authenticate decrypts signed token data to TokenClaim object and checks if token expired
func (a *jwtTokenAuthenticator) Authenticate(tokenData string) (*jwt.Claims, *TokenClaim, error) {

	tok, err := jwt.ParseSigned(tokenData)
	if err != nil {
		return nil, nil, err
	}

	public := &jwt.Claims{}
	customClaims := &TokenClaim{}

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
