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

package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func ValidateRandomSecret(value string, path string) error {
	if value == "" {
		secret, err := randomString()
		if err == nil {
			return fmt.Errorf("%s must be a non-empty secret, for example: %s", path, secret)
		}

		return fmt.Errorf("%s must be a non-empty secret", path)
	}

	return nil
}

func randomString() (string, error) {
	c := 32
	b := make([]byte, c)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
