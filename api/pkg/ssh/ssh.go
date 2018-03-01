package ssh

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	// DefaultUserLabel is the name of the user label on the ssh key cr
	DefaultUserLabel = "kubermatic-user-hash"
)

// UserListLabelSelector returns a label selector for the given user id
func UserListLabelSelector(userID string) (labels.Selector, error) {
	req, err := labels.NewRequirement(DefaultUserLabel, selection.Equals, []string{UserToLabel(userID)})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*req), nil
}

// UserToLabel encodes an arbitrary user string into a Kubernetes label value
// compatible value. This is never decoded again. It shall be without
// collisions, i.e. no hash. This is a one-way-function!
// When the user is to long it will be hashed.
// This is done for backwards compatibility!
func UserToLabel(user string) string {
	if user == "" {
		return user
	}
	// This part has to stay for backwards capability.
	// It we need this for old clusters which use an auth provider with useres, which will encode
	// in less then 63 chars.
	b := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(user))
	if len(b) <= 63 {
		return b
	}

	// This is the new way.
	// We can use a weak hash because we trust the authority, which generates the name.
	// This will always yield a string which makes the user identifiable and is less than 63 chars
	// due to the usage of a hash function.
	// Potentially we could have collisions, but this is not avoidable, due to the fact that the
	// set of our domain is smaller than our codomain.
	// It's trivial to see that we can't reverse this due to the fact that our function is not injective. q.e.d
	sh := sha1.New()
	fmt.Fprint(sh, user)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
}

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
		return nil, err
	}
	sshKeyHash := ssh.FingerprintLegacyMD5(pubKey)
	// Construct a key with the name containing the hash fragment for people to recognize it faster.
	keyName := fmt.Sprintf("key-%s-%s", strings.NewReplacer(":", "").Replace(sshKeyHash), uuid.ShortUID(4))
	userSSHKey := &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:   keyName,
			Labels: map[string]string{DefaultUserLabel: UserToLabel(sb.owner)},
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
