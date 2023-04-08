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

package grafana

import "testing"

func TestDefaultOrgID(t *testing.T) {
	// If Grafana ever changes its internal ID for the default org from 1 to
	// anything else, the ID computation magic (len(orgs)+1) in the fake
	// grafana client will not work anymore (if the default is now 3, the
	// computed ID will not match DefaultOrgID anymore).
	// Please adjust the unit tests as needed.
	if DefaultOrgID != 1 {
		t.Fatal("If the default ID is not 1 anymore, the assumptions in fakeGrafana.createDefaultOrg do not work anymore.")
	}
}
