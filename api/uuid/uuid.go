package uuid

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"time"
)

const (
	alphabet64 = "abcdefghijklmnopqrstuvwxyz01234567890"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// UUID is a very simple random uuid generator used for faking.
func UUID() (string, error) {
	b := make([]byte, 2)

	_, err := crand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b[0:2]), nil
}

// ShortUID generates a non-cryptographic random string in base62.
// Size defines the bit length of the generated uuid
func ShortUID(size int) string {
	s := make([]byte, size)
	for i := 0; i < size; i++ {
		s[i] = alphabet64[rand.Intn(len(alphabet64))]
	}
	return string(s)
}
