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
