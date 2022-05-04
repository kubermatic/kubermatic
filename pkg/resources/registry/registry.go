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

// Package registry groups all container registry related types and helpers in one place.
package registry

// WithOverwriteFunc is a function that takes a string and either returns that string or a defined override value.
type WithOverwriteFunc func(string) string

// GetOverwriteFunc returns a WithOverwriteFunc based on the given override value.
func GetOverwriteFunc(overwriteRegistry string) WithOverwriteFunc {
	if overwriteRegistry != "" {
		return func(string) string {
			return overwriteRegistry
		}
	}
	return func(s string) string {
		return s
	}
}
