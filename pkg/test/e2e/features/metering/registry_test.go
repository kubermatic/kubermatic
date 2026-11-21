/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package metering

import (
	"testing"
)

func TestFeatures(t *testing.T) {
	validScopes := map[Scope]bool{ScopeSeed: true, ScopeSeedWide: true, ScopeDisruptive: true}
	validModes := map[Mode]bool{ModeParallel: true, ModeSerial: true, ModeLast: true}

	ids := map[string]bool{}
	lastCount := 0

	for _, feature := range Features {
		if feature.ID == "" || feature.Name == "" {
			t.Errorf("feature %q needs an ID and a name", feature.ID)
		}

		if ids[feature.ID] {
			t.Errorf("feature ID %q is used more than once", feature.ID)
		}
		ids[feature.ID] = true

		if !validScopes[feature.Scope] {
			t.Errorf("feature %s has invalid scope %q", feature.ID, feature.Scope)
		}

		if !validModes[feature.Mode] {
			t.Errorf("feature %s has invalid mode %q", feature.ID, feature.Mode)
		}

		if feature.Check == nil {
			t.Errorf("feature %s has no check", feature.ID)
		}

		if feature.Doc == "" {
			t.Errorf("feature %s has no doc reference", feature.ID)
		}

		// a disruptive check turns metering off, anything scheduled after it
		// would fail
		if (feature.Scope == ScopeDisruptive) != (feature.Mode == ModeLast) {
			t.Errorf("feature %s: scope %q and mode %q must be used together", feature.ID, ScopeDisruptive, ModeLast)
		}

		// a check that changes the Seed or the bucket must not run next to
		// checks that read them
		if feature.Scope == ScopeSeedWide && feature.Mode == ModeParallel {
			t.Errorf("feature %s changes seed-wide state and must not run in parallel", feature.ID)
		}

		if feature.Mode == ModeLast {
			lastCount++
		}
	}

	if lastCount > 1 {
		t.Errorf("only one feature may run last, found %d", lastCount)
	}
}
