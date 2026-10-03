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

package rbac

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// TestResourcePredicates locks in the kubermatic/kubermatic#11292 regression: a
// projectResource that sets a non-empty namespace must not reconcile events for
// objects in other namespaces, even when its predicate matches.
func TestResourcePredicates(t *testing.T) {
	supportedPredicate := func(o ctrlruntimeclient.Object) bool {
		return shouldEnqueueSecret(o.GetName())
	}

	tests := []struct {
		name          string
		resource      projectResource
		objectName    string
		objectNS      string
		expectMatches bool
	}{
		{
			name: "scoped namespace blocks Secret in third-party namespace",
			resource: projectResource{
				object:    &corev1.Secret{},
				namespace: "kubermatic",
				predicate: supportedPredicate,
			},
			objectName:    "credentials-test",
			objectNS:      "test",
			expectMatches: false,
		},
		{
			name: "scoped namespace allows Secret in kubermatic namespace",
			resource: projectResource{
				object:    &corev1.Secret{},
				namespace: "kubermatic",
				predicate: supportedPredicate,
			},
			objectName:    "credentials-test",
			objectNS:      "kubermatic",
			expectMatches: true,
		},
		{
			name: "empty resource.namespace is cluster-scoped",
			resource: projectResource{
				object:    &corev1.Secret{},
				namespace: "",
				predicate: supportedPredicate,
			},
			objectName:    "credentials-test",
			objectNS:      "test",
			expectMatches: true,
		},
		{
			name: "predicate still excludes unsupported name prefixes in scoped namespace",
			resource: projectResource{
				object:    &corev1.Secret{},
				namespace: "kubermatic",
				predicate: supportedPredicate,
			},
			objectName:    "unrelated-secret",
			objectNS:      "kubermatic",
			expectMatches: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: tc.objectName, Namespace: tc.objectNS},
			}
			preds := resourcePredicates(tc.resource)
			matches := true
			for _, p := range preds {
				if !p.Create(event.CreateEvent{Object: obj}) {
					matches = false
					break
				}
			}
			if matches != tc.expectMatches {
				t.Errorf("predicate chain matches=%v, want %v", matches, tc.expectMatches)
			}
		})
	}
}
