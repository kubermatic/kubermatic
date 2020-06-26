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
