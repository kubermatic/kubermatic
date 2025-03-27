/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package helper

import (
	"fmt"
	"strings"
)

const (
	UserServiceAccountPrefix = "serviceaccount-"
)

// IsProjectServiceAccount determines whether the given email address
// or user object name belongs to a project service account. For a service
// account, they must have the UserServiceAccountPrefix.
func IsProjectServiceAccount(nameOrEmail string) bool {
	return strings.HasPrefix(nameOrEmail, UserServiceAccountPrefix)
}

// RemoveProjectServiceAccountPrefix removes "serviceaccount-" from a SA's ID,
// for example given "serviceaccount-7d4b5695vb" it returns "7d4b5695vb".
func RemoveProjectServiceAccountPrefix(str string) string {
	return strings.TrimPrefix(str, UserServiceAccountPrefix)
}

// EnsureProjectServiceAccountPrefix adds "serviceaccount-" prefix to a SA's ID,
// for example given "7d4b5695vb" it returns "serviceaccount-7d4b5695vb".
func EnsureProjectServiceAccountPrefix(str string) string {
	if !IsProjectServiceAccount(str) {
		return fmt.Sprintf("%s%s", UserServiceAccountPrefix, str)
	}

	return str
}
