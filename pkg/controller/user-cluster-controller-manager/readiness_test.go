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
	"errors"
	"net/http"
	"testing"
)

func TestLeaderOnlyCheck(t *testing.T) {
	notReconciled := func(_ *http.Request) error {
		return errors.New("no successful reconciliation so far")
	}

	elected := make(chan struct{})
	check := LeaderOnlyCheck(elected, notReconciled)

	if err := check(nil); err != nil {
		t.Fatalf("expected check to pass on a standby replica, got: %v", err)
	}

	close(elected)
	if err := check(nil); err == nil {
		t.Fatal("expected check to fail on the leader before it has reconciled")
	}

	reconciled := LeaderOnlyCheck(elected, func(_ *http.Request) error { return nil })
	if err := reconciled(nil); err != nil {
		t.Fatalf("expected check to pass on the leader after it has reconciled, got: %v", err)
	}
}
