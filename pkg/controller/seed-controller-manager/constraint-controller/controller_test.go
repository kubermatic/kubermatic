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

package seedconstraintsynchronizer

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	constraintName = "constraint"
	kind           = "RequiredLabel"
)

func TestReconcile(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	seedNamespace := "namespace"
	clusterNamespace := generator.GenDefaultCluster().Status.NamespaceName

	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedConstraint   *kubermaticv1.Constraint
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: sync constraint to user cluster ns",
			namespacedName: types.NamespacedName{
				Namespace: seedNamespace,
				Name:      constraintName,
			},
			expectedConstraint: genConstraint(constraintName, clusterNamespace, kind, true, false),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraint(constraintName, seedNamespace, kind, false, false), genCluster(true)).
				Build(),
		},
		{
			name: "scenario 2: dont sync constraint to user cluster ns which has opa-integration off",
			namespacedName: types.NamespacedName{
				Namespace: seedNamespace,
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraint(constraintName, seedNamespace, kind, false, false), genCluster(false)).
				Build(),
		},
		{
			name: "scenario 3: cleanup constraint on user cluster ns when seed constraint is being terminated",
			namespacedName: types.NamespacedName{
				Namespace: seedNamespace,
				Name:      constraintName,
			},
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(
					genConstraint(constraintName, seedNamespace, kind, false, true),
					genConstraint(constraintName, clusterNamespace, kind, true, false),
					genCluster(true),
				).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				namespace:               seedNamespace,
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				seedClient:              tc.seedClient,
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			constraint := &kubermaticv1.Constraint{}
			err = tc.seedClient.Get(ctx, types.NamespacedName{Name: tc.namespacedName.Name, Namespace: clusterNamespace}, constraint)

			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got constraint: %v", tc.expectedGetErrStatus, constraint)
				}
				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("failed get constraint: %v", err)
			}

			// set resource version to empty as it messes up tests
			constraint.ResourceVersion = ""

			if !diff.SemanticallyEqual(tc.expectedConstraint, constraint) {
				t.Fatalf("diff:\n%v", diff.ObjectDiff(tc.expectedConstraint, constraint))
			}
		})
	}
}

func genConstraint(name, namespace, kind string, label, deleted bool) *kubermaticv1.Constraint {
	constraint := generator.GenConstraint(name, namespace, kind)
	if label {
		if constraint.Labels != nil {
			constraint.Labels[key] = constraint.Name
		} else {
			constraint.Labels = map[string]string{key: constraint.Name}
		}
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		constraint.DeletionTimestamp = &deleteTime
		constraint.Finalizers = append(constraint.Finalizers, cleanupFinalizer)
	}
	return constraint
}

func genCluster(opaEnabled bool) *kubermaticv1.Cluster {
	cluster := generator.GenDefaultCluster()
	cluster.Spec.OPAIntegration = &kubermaticv1.OPAIntegrationSettings{
		Enabled: opaEnabled,
	}
	cluster.Spec.Cloud.Fake = nil
	cluster.Spec.Cloud.AWS = &kubermaticv1.AWSCloudSpec{}
	cluster.Labels = map[string]string{"deployment": "prod", "domain": "sales", "cluster": "test"}
	return cluster
}
