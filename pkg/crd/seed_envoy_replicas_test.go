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

package crd

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestSeedEnvoyReplicasHasNoCRDDefault guards against re-introducing a CRD
// default for spec.nodeportProxy.envoy.replicas. API servers apply CRD defaults
// on every write, and because the seed-sync controller always serializes the
// envoy object, a default here causes the master and seed copies to differ
// forever (see issue #16862). The operator applies the default of 3 itself.
func TestSeedEnvoyReplicasHasNoCRDDefault(t *testing.T) {
	crd, err := CRDForGVK(schema.GroupVersionKind{Group: "kubermatic.k8c.io", Version: "v1", Kind: "Seed"})
	if err != nil {
		t.Fatalf("failed to load seed CRD: %v", err)
	}

	for _, version := range crd.Spec.Versions {
		spec := version.Schema.OpenAPIV3Schema.Properties["spec"]
		replicas, ok := spec.Properties["nodeportProxy"].Properties["envoy"].Properties["replicas"]
		if !ok {
			t.Fatalf("version %s: spec.nodeportProxy.envoy.replicas not found in schema", version.Name)
		}
		if replicas.Default != nil {
			t.Errorf("version %s: spec.nodeportProxy.envoy.replicas must not have a CRD default, got default=%s", version.Name, replicas.Default.Raw)
		}
	}
}
