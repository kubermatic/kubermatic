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

package userclustercontrollermanager

import (
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// LeaderOnlyCheck wraps a readiness check that can only pass on the leader,
// such as a "reconciled successfully once" check for a controller that only
// runs while holding the leader lease. Until elected is closed, the replica is
// a standby and the check passes, so standby replicas become Ready. Once the
// replica is elected, check decides readiness as before.
func LeaderOnlyCheck(elected <-chan struct{}, check healthz.Checker) healthz.Checker {
	return func(req *http.Request) error {
		select {
		case <-elected:
			return check(req)
		default:
			return nil
		}
	}
}
