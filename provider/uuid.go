package provider

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
)

// UUID is a very simple random uuid generator used for faking.
func UUID() (string, error) {
	b := make([]byte, 2)

	_, err := crand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b[0:2]), nil
}

const (
	alphabet64 = "abcdefghijklmnopqrstuvwxyz01234567890"
)

// ShortUID generates a non-cryptographic random string in base62.
func ShortUID(size int) string {
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = alphabet64[rand.Intn(len(alphabet64))]
	}
	return string(s)
}
