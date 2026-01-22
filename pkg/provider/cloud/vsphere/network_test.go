//go:build integration

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

package vsphere

import (
	"context"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestGetPossibleVMNetworks(t *testing.T) {
	sim := vSphereSimulator{t: t}
	sim.setUp()
	defer sim.tearDown()

	dc := &kubermaticv1.DatacenterSpecVSphere{}
	sim.fillClientInfo(dc)

	networks, err := GetNetworks(context.Background(), dc, sim.username(), sim.password(), nil)
	if err != nil {
		t.Fatalf("GetNetworks failed: %v", err)
	}

	if len(networks) == 0 {
		t.Fatal("expected at least one network, got none")
	}

	const expectedPathPrefix = "/DC0/network/"
	for _, n := range networks {
		if n.Name == "" {
			t.Error("network Name should not be empty")
		}
		if n.AbsolutePath == "" {
			t.Error("network AbsolutePath should not be empty")
		}
		if n.RelativePath == "" {
			t.Error("network RelativePath should not be empty")
		}
		if n.Type == "" {
			t.Error("network Type should not be empty")
		}
		if !strings.HasPrefix(n.AbsolutePath, expectedPathPrefix) {
			t.Errorf("expected AbsolutePath to start with %q, got %s", expectedPathPrefix, n.AbsolutePath)
		}
	}
}
