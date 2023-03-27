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

package test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// AssertContainsExactly failed the test if the actual slice does not contain exactly the element of the expected slice.
func AssertContainsExactly[T comparable](t *testing.T, prefixMsg string, actual []T, expected []T) {
	t.Helper()

	missing := map[T]struct{}{}
	notExpected := map[T]struct{}{}
	for _, val := range expected {
		missing[val] = struct{}{}
	}

	for _, val := range actual {
		notExpected[val] = struct{}{}
	}

	for _, val := range actual {
		if _, found := missing[val]; found {
			delete(missing, val)
			delete(notExpected, val)
		}
	}
	if len(missing) != 0 || len(notExpected) != 0 {
		t.Fatalf("%s. expect %+v, to contains only %+v.\nMissing elements %+v\nUnexpected elements %+v", prefixMsg, actual, expected, keys(missing), keys(notExpected))
	}
}

// keys return the keys of the map.
func keys[T comparable](dict map[T]struct{}) []T {
	var res []T
	for k := range dict {
		res = append(res, k)
	}
	return res
}

// ReleaseStorageInfo holds information about the secret storing the Helm release information.
type ReleaseStorageInfo struct {
	// Name of the secret containing release information.
	Name string

	// Version of the release.
	Version string
}

// MapToReleaseStorageInfo maps the secrets containing the Helm release information to a smaller struct only containing the Name of the secret and the release version.
func MapToReleaseStorageInfo(secrets []corev1.Secret) []ReleaseStorageInfo {
	res := []ReleaseStorageInfo{}
	for _, secret := range secrets {
		res = append(res, ReleaseStorageInfo{
			Name:    secret.Name,
			Version: secret.Labels["version"],
		})
	}
	return res
}
