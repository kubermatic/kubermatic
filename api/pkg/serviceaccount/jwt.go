package serviceaccount

import (
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type TokenGenerator interface {
	// GenerateToken generates a token which will identify the given
	// ServiceAccount. privateClaims is an interface that will be
	// serialized into the JWT payload JSON encoding at the root level of
	// the payload object. Public claims take precedent over private
	// claims i.e. if both claims and privateClaims have an "exp" field,
	// the value in claims will be used.
	GenerateToken(claims *jwt.Claims, privateClaims *TokenClaim) (string, error)
}

type TokenAuthenicator interface {
	// AuthenticateToken checks given token and transform it to private claim object
	AuthenticateToken(tokenData string) (*TokenClaim, error)
}

type TokenClaim struct {
	Email     string `json:"email,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	TokenID   string `json:"token_id,omitempty"`
}

func Claims(email, projectID, tokenID string) (*jwt.Claims, *TokenClaim) {

	now := time.Now
	sc := &jwt.Claims{
		IssuedAt:  jwt.NewNumericDate(now()),
		NotBefore: jwt.NewNumericDate(now()),
		Expiry:    jwt.NewNumericDate(now().AddDate(3, 0, 0)),
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

// GenerateToken generates new token from claims
func (j *jwtTokenGenerator) GenerateToken(claims *jwt.Claims, privateClaims *TokenClaim) (string, error) {
	// claims are applied in reverse precedence
	return jwt.Signed(j.signer).
		Claims(privateClaims).
		Claims(claims).
		CompactSerialize()
}

// JWTTokenAuthenticator authenticates tokens as JWT tokens produced by JWTTokenGenerator
func JWTTokenAuthenticator(privateKey []byte) TokenAuthenicator {
	return &jwtTokenAuthenticator{
		key: privateKey,
	}
}

// AuthenticateToken decrypts signed token data to TokenClaim object and checks if token expired
func (a *jwtTokenAuthenticator) AuthenticateToken(tokenData string) (*TokenClaim, error) {

	tok, err := jwt.ParseSigned(tokenData)
	if err != nil {
		return nil, err
	}

	public := &jwt.Claims{}
	private := &TokenClaim{}

	if err := tok.Claims(a.key, private, public); err != nil {
		return nil, err
	}

	err = public.Validate(jwt.Expected{
		Time: time.Now(),
	})
	switch {
	case err == nil:
	case err == jwt.ErrExpired:
		return nil, fmt.Errorf("token has expired")
	default:
		return nil, fmt.Errorf("token could not be validated due to error: %v", err)
	}

	return private, nil
}
