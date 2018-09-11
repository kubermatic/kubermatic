package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/pborman/uuid"
)

const privateRSAKeyBitSize = 4096

// Pubkey is only used to create temporary keypairs, thus we
// do not need the Private key
// The reason for not hardcoding a random public key is that
// it would look like a backdoor
type Pubkey struct {
	Name           string
	PublicKey      string
	FingerprintMD5 string
}

func NewKey() (*Pubkey, error) {
	tmpRSAKeyPair, err := rsa.GenerateKey(rand.Reader, privateRSAKeyBitSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create private RSA key: %v", err)
	}

	if err := tmpRSAKeyPair.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate private RSA key: %v", err)
	}

	pubKey, err := ssh.NewPublicKey(&tmpRSAKeyPair.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ssh public key: %v", err)
	}

	return &Pubkey{
		Name:           uuid.New(),
		PublicKey:      string(ssh.MarshalAuthorizedKey(pubKey)),
		FingerprintMD5: ssh.FingerprintLegacyMD5(pubKey),
	}, nil
}
