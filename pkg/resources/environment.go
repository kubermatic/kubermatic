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

package resources

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// SanitizeEnvVar will take the value of an environment variable and sanitize it.
// the need for this comes from github.com/kubermatic/kubermatic/issues/7960.
func SanitizeEnvVars(envVars []corev1.EnvVar) []corev1.EnvVar {
	for idx := range envVars {
		envVars[idx].Value = strings.ReplaceAll(envVars[idx].Value, "$", "$$")
	}

	return envVars
}
