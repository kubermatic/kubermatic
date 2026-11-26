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

func TestResourcePredicatesHonorNamespace(t *testing.T) {
	secretPredicate := func(o ctrlruntimeclient.Object) bool {
		return shouldEnqueueSecret(o.GetName())
	}

	testCases := []struct {
		name      string
		resource  projectResource
		object    ctrlruntimeclient.Object
		wantAdmit bool
	}{
		{
			name:      "matching secret in the configured namespace is watched",
			resource:  projectResource{namespace: "kubermatic", predicate: secretPredicate},
			object:    &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic", Name: "credentials-test"}},
			wantAdmit: true,
		},
		{
			name:      "matching secret in another namespace is ignored",
			resource:  projectResource{namespace: "kubermatic", predicate: secretPredicate},
			object:    &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "credentials-test"}},
			wantAdmit: false,
		},
		{
			name:      "non-matching secret in the configured namespace is ignored",
			resource:  projectResource{namespace: "kubermatic", predicate: secretPredicate},
			object:    &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "kubermatic", Name: "unrelated"}},
			wantAdmit: false,
		},
		{
			name:      "resource without namespace is watched in every namespace",
			resource:  projectResource{predicate: secretPredicate},
			object:    &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "credentials-test"}},
			wantAdmit: true,
		},
		{
			name:      "resource without predicate is limited by namespace only",
			resource:  projectResource{namespace: "kubermatic"},
			object:    &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "anything"}},
			wantAdmit: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			admitted := true
			for _, p := range resourcePredicates(tc.resource) {
				admitted = admitted &&
					p.Create(event.CreateEvent{Object: tc.object}) &&
					p.Update(event.UpdateEvent{ObjectOld: tc.object, ObjectNew: tc.object}) &&
					p.Delete(event.DeleteEvent{Object: tc.object})
			}

			if admitted != tc.wantAdmit {
				t.Fatalf("expected watched=%v, got %v", tc.wantAdmit, admitted)
			}
		})
	}
}
