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

package openshift

import (
	"testing"
)

func TestAESRoundTripping(t *testing.T) {
	valueToEncrypt := []byte("my-very-secret-text")
	psk := []byte("8w6xrx.89vwtn8strwcwbzt")

	ciphertext, err := aesEncrypt(valueToEncrypt, psk)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}
	plaintext, err := AESDecrypt(ciphertext, psk)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if string(valueToEncrypt) != string(plaintext) {
		t.Fatalf("Result %q does not match initial value %q", string(valueToEncrypt), string(plaintext))
	}
}
