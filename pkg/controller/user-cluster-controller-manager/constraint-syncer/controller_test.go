//go:build ee

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

package constraintsyncer

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	constraintName = "constraint"
	kind           = "RequiredLabel"
)

func TestReconcile(t *testing.T) {
	testScheme := fake.NewScheme()
	utilruntime.Must(test.GatekeeperSchemeBuilder.AddToScheme(testScheme))

	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedConstraint   apiv2.Constraint
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
		userClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: sync constraint with rawJSON to user cluster",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: generator.GenDefaultAPIConstraint(constraintName, kind),
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(
					func() ctrlruntimeclient.Object {
						c := generator.GenConstraint(constraintName, "namespace", kind)
						c.Spec.Parameters = map[string]json.RawMessage{
							"rawJSON": []byte(`"{\"labels\":[\"gatekeeper\",\"opa\"]}"`),
						}
						return c
					}()).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build(),
		},
		{
			name: "scenario 2: cleanup gatekeeper constraint on user cluster when kubermatic constraint on seed cluster is being terminated",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(func() *kubermaticv1.Constraint {
					c := generator.GenConstraint(constraintName, "namespace", kind)
					deleteTime := metav1.NewTime(time.Now())
					c.DeletionTimestamp = &deleteTime
					c.Finalizers = []string{kubermaticv1.GatekeeperConstraintCleanupFinalizer}
					return c
				}()).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(&test.RequiredLabel{
					ObjectMeta: metav1.ObjectMeta{
						Name: constraintName,
					},
				}).
				Build(),
		},
		{
			name: "scenario 3: delete kubermatic constraint on seed cluster when the corresponding constraint on user cluster is missing",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(func() *kubermaticv1.Constraint {
					c := generator.GenConstraint(constraintName, "namespace", kind)
					deleteTime := metav1.NewTime(time.Now())
					c.DeletionTimestamp = &deleteTime
					c.Finalizers = []string{kubermaticv1.GatekeeperConstraintCleanupFinalizer}
					return c
				}()).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build(),
		},
		{
			name: "scenario 4: sync constraint to user cluster",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: generator.GenDefaultAPIConstraint(constraintName, kind),
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(generator.GenConstraint(constraintName, "namespace", kind)).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build(),
		},
		{
			name: "scenario 5: don't create kubermatic constraint on user cluster when the corresponding constraint on user cluster ns is disabled",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(func() *kubermaticv1.Constraint {
					c := generator.GenConstraint(constraintName, "namespace", kind)
					c.Spec.Disabled = true
					return c
				}()).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build(),
		},
		{
			name: "scenario 6: delete kubermatic constraint on user cluster when the corresponding constraint on user cluster ns is disabled",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(func() *kubermaticv1.Constraint {
					c := generator.GenConstraint(constraintName, "namespace", kind)
					c.Spec.Disabled = true
					return c
				}()).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(&test.RequiredLabel{
					ObjectMeta: metav1.ObjectMeta{
						Name: constraintName,
					},
				}).
				Build(),
		},
		{
			name: "scenario 7: sync constraint to user cluster with enforcement Action defined",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: func() apiv2.Constraint {
				constraint := generator.GenDefaultAPIConstraint(constraintName, kind)
				constraint.Spec.EnforcementAction = "deny"
				return constraint
			}(),
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(
					func() *kubermaticv1.Constraint {
						constraint := generator.GenConstraint(constraintName, "namespace", kind)
						constraint.Spec.EnforcementAction = "deny"
						return constraint
					}(),
				).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build(),
		},
		{
			name: "scenario 8: update constraint to user cluster with enforcement Action removed",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: func() apiv2.Constraint {
				constraint := generator.GenDefaultAPIConstraint(constraintName, kind)
				return constraint
			}(),
			seedClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(
					func() *kubermaticv1.Constraint {
						constraint := generator.GenConstraint(constraintName, "namespace", kind)
						return constraint
					}(),
				).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(&test.RequiredLabel{
					ObjectMeta: metav1.ObjectMeta{
						Name: constraintName,
					},
					Spec: test.ConstraintSpec{EnforcementAction: "warn"},
				}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:        kubermaticlog.Logger,
				recorder:   &events.FakeRecorder{},
				seedClient: tc.seedClient,
				userClient: tc.userClient,
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			reqLabel := &unstructured.Unstructured{}
			reqLabel.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "constraints.gatekeeper.sh",
				Version: "v1beta1",
				Kind:    kind,
			})
			err := tc.userClient.Get(ctx, types.NamespacedName{Name: constraintName}, reqLabel)
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got constraint: %v", tc.expectedGetErrStatus, reqLabel)
				}

				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get constraint: %v", err)
			}

			// get match
			matchMap, err := unmarshalToJSONMap(tc.expectedConstraint.Spec.Match)
			if err != nil {
				t.Fatalf("failed to unmarshal expected match: %v", err)
			}
			resultMatch, found, err := unstructured.NestedFieldNoCopy(reqLabel.Object, "spec", "match")
			if err != nil || !found {
				t.Fatalf("failed to get nested match field (found %t): %v", found, err)
			}

			resultParams, found, err := unstructured.NestedFieldNoCopy(reqLabel.Object, "spec", "parameters")
			if err != nil || !found {
				t.Fatalf("failed to get nested params field (found %t): %v", found, err)
			}

			// compare
			if !diff.SemanticallyEqual(matchMap, resultMatch) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(matchMap, resultMatch))
			}
			// cast params to bytes for comparison
			expectedParamsBytes, err := json.Marshal(tc.expectedConstraint.Spec.Parameters)
			if err != nil {
				t.Fatal(err)
			}

			resultParamsBytes, err := json.Marshal(resultParams)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expectedParamsBytes, resultParamsBytes) {
				t.Fatalf("Parameters differ:\n%v", diff.ObjectDiff(tc.expectedConstraint.Spec.Parameters, resultParamsBytes))
			}

			if reqLabel.GetName() != tc.expectedConstraint.Name {
				t.Fatalf("Expected name %q, got %q", tc.expectedConstraint.Name, reqLabel.GetName())
			}

			// Check EnforcementAction is set with expected value
			resultEnforcementAction, found, err := unstructured.NestedFieldNoCopy(reqLabel.Object, spec, enforcementAction)
			if err != nil {
				t.Fatalf("failed to get nested enforcementAction field: %v", err)
			}
			if !found {
				resultEnforcementAction = ""
			}
			if tc.expectedConstraint.Spec.EnforcementAction != resultEnforcementAction {
				t.Fatalf("Expected .spec.enforcementAction '%s', got '%s'", tc.expectedConstraint.Spec.EnforcementAction, resultEnforcementAction)
			}
		})
	}
}
