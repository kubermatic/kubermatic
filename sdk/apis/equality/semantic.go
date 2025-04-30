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

package equality

import (
	"time"

	semverlib "github.com/Masterminds/semver/v3"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/conversion"
)

// Semantic can do semantic deep equality checks for objects.
// Example: equality.Semantic.DeepEqual(aPod, aPodWithNonNilButEmptyMaps) == true.
var Semantic = conversion.EqualitiesOrDie(
	func(a, b resource.Quantity) bool {
		return a.Cmp(b) == 0
	},
	func(a, b *semverlib.Version) bool {
		if a == nil && b == nil {
			return true
		}

		if a != nil && b != nil {
			return a.Equal(b)
		}

		return false
	},
	func(a, b time.Time) bool {
		return a.Equal(b)
	},
)
