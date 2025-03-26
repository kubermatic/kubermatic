/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package defaulting

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func DefaultUserSSHKey(key *kubermaticv1.UserSSHKey, oldKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error) {
	if key.Spec.PublicKey == "" {
		return nil, errors.New("spec.publicKey cannot be empty")
	}

	if oldKey == nil || (oldKey.Spec.PublicKey != key.Spec.PublicKey) || (oldKey.Spec.Fingerprint != key.Spec.Fingerprint) {
		// parse the key
		pubKeyParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.Spec.PublicKey))
		if err != nil {
			return nil, fmt.Errorf("the provided SSH key is invalid: %w", err)
		}

		// calculate the fingerprint
		key.Spec.Fingerprint = ssh.FingerprintLegacyMD5(pubKeyParsed)
	}

	return key, nil
}
