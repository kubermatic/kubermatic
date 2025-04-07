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

package masterconstraintsynchronizer

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	constraintName = "constraint"
	kind           = "RequiredLabel"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedConstraint   *kubermaticv1.Constraint
		expectedGetErrStatus metav1.StatusReason
		masterClient         ctrlruntimeclient.Client
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: sync constraint to seed cluster",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: genConstraint(constraintName, "namespace", kind, false),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genConstraint(constraintName, "namespace", kind, false), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name: "scenario 2: cleanup gatekeeper constraint on seed cluster when master constraint is being terminated",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genConstraint(constraintName, "namespace", kind, true), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraint(constraintName, "namespace", kind, false)).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			constraint := &kubermaticv1.Constraint{}
			err := tc.seedClient.Get(ctx, types.NamespacedName{Name: constraintName}, constraint)
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got ct: %v", tc.expectedGetErrStatus, constraint)
				}

				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get constraint: %v", err)
			}

			constraint.ResourceVersion = tc.expectedConstraint.ResourceVersion
			constraint.Namespace = tc.expectedConstraint.Namespace

			if !diff.SemanticallyEqual(tc.expectedConstraint, constraint) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedConstraint, constraint))
			}
		})
	}
}

func genConstraint(name, namespace, kind string, deleted bool) *kubermaticv1.Constraint {
	constraint := generator.GenConstraint(name, namespace, kind)
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		constraint.DeletionTimestamp = &deleteTime
		constraint.Finalizers = append(constraint.Finalizers, cleanupFinalizer)
	}

	return constraint
}
