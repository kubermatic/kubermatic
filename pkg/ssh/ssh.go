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

package ssh

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSSHKeyBuilder is builder to create ssh key structs including validation.
type UserSSHKeyBuilder struct {
	owner     string
	name      string
	publicKey string
}

// NewUserSSHKeyBuilder returns a new instance of a UserSSHKeyBuilder.
func NewUserSSHKeyBuilder() *UserSSHKeyBuilder {
	return &UserSSHKeyBuilder{}
}

// SetName sets the name for a ssh key.
func (sb *UserSSHKeyBuilder) SetName(keyName string) *UserSSHKeyBuilder {
	sb.name = keyName
	return sb
}

// SetRawKey sets the raw public key for a ssh key.
func (sb *UserSSHKeyBuilder) SetRawKey(publicKey string) *UserSSHKeyBuilder {
	sb.publicKey = publicKey
	return sb
}

// SetOwner sets the username for a ssh key.
func (sb *UserSSHKeyBuilder) SetOwner(username string) *UserSSHKeyBuilder {
	sb.owner = username
	return sb
}

// Validate returns errors if the supplied data is not valid.
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

// Build returns a instance of a ssh key.
func (sb *UserSSHKeyBuilder) Build() (*kubermaticv1.UserSSHKey, error) {
	if err := sb.Validate(); err != nil {
		return nil, fmt.Errorf("key is not valid: %w", err)
	}

	name, err := uuid.UUID()
	if err != nil {
		return nil, fmt.Errorf("failed to create name: %w", err)
	}

	userSSHKey := &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("key-%s", name),
		},
		Spec: kubermaticv1.SSHKeySpec{
			Owner:     sb.owner,
			PublicKey: sb.publicKey, // webhook will add the fingerprint later
			Name:      sb.name,
			Clusters:  []string{},
		},
	}

	return userSSHKey, nil
}
