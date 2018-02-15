package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

const privateKeyBitSize = 2048

type PrivateKey struct {
	key  *rsa.PrivateKey
	name string
}

func (p *PrivateKey) Name() string {
	return p.name
}

func (p *PrivateKey) PublicKey() rsa.PublicKey {
	return p.key.PublicKey
}

// NewPrivateKey generates a new PrivateKey
func NewPrivateKey(name string) (key *PrivateKey, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, privateKeyBitSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key: %v", err)
	}

	if err := priv.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate private key: %v", err)
	}

	return &PrivateKey{key: priv, name: name}, nil
}
