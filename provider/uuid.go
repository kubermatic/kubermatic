package provider

import (
	"crypto/rand"
	"fmt"
)

// UUID is a very simple random uuid generator used for faking.
func UUID() (string, error) {
	b := make([]byte, 2)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b[0:2]), nil
}
