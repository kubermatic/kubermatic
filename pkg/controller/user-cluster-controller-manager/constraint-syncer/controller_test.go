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
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	constrainthandler "k8c.io/kubermatic/v2/pkg/handler/v2/constraint"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	constraintName = "constraint"
	kind           = "RequiredLabel"
)

func TestReconcile(t *testing.T) {
	if err := test.RegisterScheme(test.SchemeBuilder); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedConstraint   apiv2.Constraint
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
		userClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: sync constraint to user cluster",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedConstraint: test.GenDefaultAPIConstraint(constraintName, kind),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.GenConstraint(constraintName, "namespace", kind)).
				Build(),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build(),
		},
		{
			name: "scenario 2: cleanup gatekeeper constraint on user cluster when kubermatic constraint on seed cluster is being terminated",
			namespacedName: types.NamespacedName{
				Namespace: "namespace",
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(func() *v1.Constraint {
					c := test.GenConstraint(constraintName, "namespace", kind)
					deleteTime := metav1.NewTime(time.Now())
					c.DeletionTimestamp = &deleteTime
					c.Finalizers = []string{kubermaticapiv1.GatekeeperConstraintCleanupFinalizer}
					return c
				}()).
				Build(),
			userClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(&test.RequiredLabel{
					ObjectMeta: metav1.ObjectMeta{
						Name: constraintName,
					},
				}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:        kubermaticlog.Logger,
				recorder:   &record.FakeRecorder{},
				seedClient: tc.seedClient,
				userClient: tc.userClient,
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			reqLabel := &unstructured.Unstructured{}
			reqLabel.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   constrainthandler.ConstraintsGroup,
				Version: constrainthandler.ConstraintsVersion,
				Kind:    kind,
			})
			err := tc.userClient.Get(ctx, types.NamespacedName{Name: constraintName}, reqLabel)
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got constraint: %v", tc.expectedGetErrStatus, reqLabel)
				}

				if tc.expectedGetErrStatus != errors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, errors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get constraint: %v", err)
			}

			// get match
			matchMap, err := unmarshallToJSONMap(tc.expectedConstraint.Spec.Match)
			if err != nil {
				t.Fatalf("failed to unmarshall expected match: %v", err)
			}
			expectedMatch, found, err := unstructured.NestedFieldNoCopy(reqLabel.Object, "spec", "match")
			if err != nil || !found {
				t.Fatalf("failed to get nested match field (found %t): %v", found, err)
			}

			// get params
			var paramsMap map[string]interface{}
			err = json.Unmarshal([]byte(tc.expectedConstraint.Spec.Parameters.RawJSON), &paramsMap)
			if err != nil {
				t.Fatalf("failed to unmarshall expected params: %v", err)
			}
			expectedParams, found, err := unstructured.NestedFieldNoCopy(reqLabel.Object, "spec", "parameters")
			if err != nil || !found {
				t.Fatalf("failed to get nested params field (found %t): %v", found, err)
			}

			// compare
			if !reflect.DeepEqual(matchMap, expectedMatch) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(matchMap, matchMap))
			}

			if !reflect.DeepEqual(paramsMap, expectedParams) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(paramsMap, expectedParams))
			}

			if !reflect.DeepEqual(reqLabel.GetName(), tc.expectedConstraint.Name) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(reqLabel.GetName(), tc.expectedConstraint.Name))
			}
		})
	}
}
