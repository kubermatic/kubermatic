package v1

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/provider/kubernetes/util"
	"github.com/kubermatic/kubermatic/api/uuid"
	ssh "golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SSHKeyPlural = "usersshkeies"
)

//+genclient
//+genclient:nonNamespaced

// UserSSHKey specifies a users UserSSHKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserSSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHKeySpec `json:"spec"`
}

type SSHKeySpec struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"public_key"`
	Clusters    []string `json:"clusters"`
}

func (sk *UserSSHKey) IsUsedByCluster(clustername string) bool {
	if sk.Spec.Clusters == nil {
		return false
	}
	for _, name := range sk.Spec.Clusters {
		if name == clustername {
			return true
		}
	}
	return false
}

func (sk *UserSSHKey) RemoveFromCluster(clustername string) {
	for i, cl := range sk.Spec.Clusters {
		if cl != clustername {
			continue
		}
		// Don't break we don't check for duplicates when adding clusters!
		sk.Spec.Clusters = append(sk.Spec.Clusters[:i], sk.Spec.Clusters[i+1:]...)
	}
}

func (sk *UserSSHKey) AddToCluster(clustername string) {
	sk.Spec.Clusters = append(sk.Spec.Clusters, clustername)
}

type UserSSHKeyBuilder struct {
	owner     string
	name      string
	publicKey string
}

func NewUserSSHKeyBuilder() *UserSSHKeyBuilder {
	return &UserSSHKeyBuilder{}
}

func (sb *UserSSHKeyBuilder) SetName(keyName string) *UserSSHKeyBuilder {
	sb.name = keyName
	return sb
}

func (sb *UserSSHKeyBuilder) SetRawKey(publicKey string) *UserSSHKeyBuilder {
	sb.publicKey = publicKey
	return sb
}

func (sb *UserSSHKeyBuilder) SetOwner(username string) *UserSSHKeyBuilder {
	sb.owner = username
	return sb
}

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

func (sb *UserSSHKeyBuilder) Build() (*UserSSHKey, error) {
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
	userSSHKey := &UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:   keyName,
			Labels: map[string]string{util.DefaultUserLabel: util.UserToLabel(sb.owner)},
		},
		Spec: SSHKeySpec{
			Owner:       sb.owner,
			PublicKey:   sb.publicKey,
			Fingerprint: sshKeyHash,
			Name:        sb.name,
		},
	}
	return userSSHKey, nil
}

// UserSSHKeyList specifies a users UserSSHKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserSSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserSSHKey `json:"items"`
}
