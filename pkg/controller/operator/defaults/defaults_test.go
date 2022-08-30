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

package defaults

import (
	"testing"

	"k8c.io/kubermatic/v2/pkg/semver"
)

func TestAutomaticUpdateRulesMatchVersions(t *testing.T) {
	for i, update := range DefaultKubernetesVersioning.Updates {
		// only test automatic rules
		if update.Automatic == nil || !*update.Automatic {
			continue
		}

		toVersion, err := semver.NewSemver(update.To)
		if err != nil {
			t.Errorf("Version %q in update rule %d is not a valid version: %v", update.To, i, err)
			continue
		}

		found := false
		for _, v := range DefaultKubernetesVersioning.Versions {
			if v.Equal(toVersion) {
				found = true
			}
		}

		if !found {
			t.Errorf("Version %s in update rule %d is not configured as a supported version.", update.To, i)
		}
	}
}

func TestDefaultVersionIsSupported(t *testing.T) {
	found := false
	for _, v := range DefaultKubernetesVersioning.Versions {
		if v.Equal(DefaultKubernetesVersioning.Default) {
			found = true
		}
	}

	if !found {
		t.Errorf("Default version %s is not configured as a supported version.", DefaultKubernetesVersioning.Default)
	}
}
