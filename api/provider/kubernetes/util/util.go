package util

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
)

// UserToLabel encodes an arbitrary user string into a Kubernetes label value
// compatible value. This is never decoded again. It shall be without
// collisions, i.e. no hash. This is a one-way-function!
// When the user is to long it will be hashed.
// This is done for backwards compatibility!
func UserToLabel(user string) string {
	if user == "" {
		return user
	}
	// This part has to stay for backwards capability.
	// It we need this for old clusters which use an auth provider with useres, which will encode
	// in less then 63 chars.
	b := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(user))
	if len(b) <= 63 {
		return b
	}

	// This is the new way.
	// We can use a weak hash because we trust the authority, which generates the name.
	// This will always yield a string which makes the user identifiable and is less than 63 chars
	// due to the usage of a hash function.
	// Potentially we could have collisions, but this is not avoidable, due to the fact that the
	// set of our domain is smaller than our codomain.
	// It's trivial to see that we can't reverse this due to the fact that our function is not injective. q.e.d
	sh := sha1.New()
	fmt.Fprint(sh, user)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
}
