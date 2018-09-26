package hash

import (
	"crypto/sha512"
	"fmt"
	"io"
)

const (
	// UserIDSuffix defines a static suffix to append to all user ID's. That way we can tell during a migration if its our current format
	UserIDSuffix = "_KUBE"
)

// GetUserID returns a hashed user ID which is a valid label
func GetUserID(v string) (string, error) {
	h := sha512.New512_224()
	if _, err := io.WriteString(h, v); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x%s", h.Sum(nil), UserIDSuffix), nil
}
