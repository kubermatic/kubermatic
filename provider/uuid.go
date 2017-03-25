package provider

import "github.com/kubermatic/api/uuid"

// UUID is a very simple random uuid generator used for faking.
// DEPRECATED: moved to "github.com/kubermatic/api/uuid"
func UUID() (string, error) {
	return uuid.UUID()
}

const (
	alphabet64 = "abcdefghijklmnopqrstuvwxyz01234567890"
)

// ShortUID generates a non-cryptographic random string in base62.
// DEPRECATED: moved to "github.com/kubermatic/api/uuid"
func ShortUID(size int) string {
	return uuid.ShortUID(size)
}
