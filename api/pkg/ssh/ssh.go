package ssh

import (
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSSHKeyBuilder is builder to create ssh key structs including validation
type UserSSHKeyBuilder struct {
	owner     string
	name      string
	publicKey string
}

// NewUserSSHKeyBuilder returns a new instance of a UserSSHKeyBuilder
func NewUserSSHKeyBuilder() *UserSSHKeyBuilder {
	return &UserSSHKeyBuilder{}
}

// SetName sets the name for a ssh key
func (sb *UserSSHKeyBuilder) SetName(keyName string) *UserSSHKeyBuilder {
	sb.name = keyName
	return sb
}

// SetRawKey sets the raw public key for a ssh key
func (sb *UserSSHKeyBuilder) SetRawKey(publicKey string) *UserSSHKeyBuilder {
	sb.publicKey = publicKey
	return sb
}

// SetOwner sets the username for a ssh key
func (sb *UserSSHKeyBuilder) SetOwner(username string) *UserSSHKeyBuilder {
	sb.owner = username
	return sb
}

// Validate returns errors if the supplied data is not valid
func (sb *UserSSHKeyBuilder) Validate() error {
	if sb.name == "" {
		return fmt.Errorf("name is missing but required")
	}
	if sb.publicKey == "" {
		return fmt.Errorf("publickey is missing but required")
	}
	if sb.owner == "" {
		return fmt.Errorf("owner is missing but required")
	}
	return nil
}

// Build returns a instance of a ssh key
func (sb *UserSSHKeyBuilder) Build() (*kubermaticv1.UserSSHKey, error) {
	if err := sb.Validate(); err != nil {
		return nil, fmt.Errorf("key is not valid: %v", err)
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(sb.publicKey))
	if err != nil {
		return nil, fmt.Errorf("the provided ssh key is invalid due to = %v", err)
	}
	sshKeyHash := ssh.FingerprintLegacyMD5(pubKey)
	// Construct a key with the name containing the hash fragment for people to recognize it faster.
	keyName := fmt.Sprintf("key-%s-%s", strings.NewReplacer(":", "").Replace(sshKeyHash), uuid.ShortUID(4))
	userSSHKey := &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: keyName,
		},
		Spec: kubermaticv1.SSHKeySpec{
			Owner:       sb.owner,
			PublicKey:   sb.publicKey,
			Fingerprint: sshKeyHash,
			Name:        sb.name,
			Clusters:    []string{},
		},
	}
	return userSSHKey, nil
}
