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

package v1

import corev1 "k8s.io/api/core/v1"

const (
	// ApplicationDefinitionSeedCleanupFinalizer indicates that synced application definition on seed clusters need cleanup.
	ApplicationDefinitionSeedCleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-application-definition"
)

// GlobalSecretKeySelector is needed as we can not use v1.SecretKeySelector
// because it is not cross namespace.
type GlobalSecretKeySelector struct {
	corev1.SecretReference `json:",inline"`
	Key                    string `json:"key"`
}
