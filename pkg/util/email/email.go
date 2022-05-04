/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package email

import (
	"fmt"
	"strings"
)

func normalizeEmail(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid email, must be of form user@domain")
	}

	parts[1] = strings.ToLower(parts[1])

	return strings.Join(parts, "@"), nil
}

func MatchesRequirements(email string, requiredEmails []string) (bool, error) {
	// no restrictions :)
	if len(requiredEmails) == 0 {
		return true, nil
	}

	normalized, err := normalizeEmail(email)
	if err != nil {
		return false, err
	}

	split := strings.Split(normalized, "@")
	emailDomain := split[1]
	matches := false

	for _, required := range requiredEmails {
		split := strings.Split(required, "@")

		switch len(split) {
		// only a domain was configured as a requirement
		case 1:
			if strings.EqualFold(split[0], emailDomain) {
				matches = true
			}

		// a full email was configured
		case 2:
			// perform the same normalization as we did on the input
			required, err = normalizeEmail(required)
			if err != nil {
				return false, fmt.Errorf("invalid email requirement %q: %w", required, err)
			}

			// no EqualFold here; because both emails are normalized, the
			// domain part is lowercased already, and we do not want to
			// ignore case differences in the user part, by design (
			// i.e. "USER@example.com" and "user@example.com" are different).
			if required == normalized {
				matches = true
			}

		// invalid configuration
		default:
			return false, fmt.Errorf("invalid email requirement %q", required)
		}
	}

	return matches, nil
}
