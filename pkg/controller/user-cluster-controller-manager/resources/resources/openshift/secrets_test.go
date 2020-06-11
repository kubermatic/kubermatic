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
