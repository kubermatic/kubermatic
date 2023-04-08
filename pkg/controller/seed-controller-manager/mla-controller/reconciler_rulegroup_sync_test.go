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

package mlacontroller

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/test/generator"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestRuleGroupSyncReconciler(objects []ctrlruntimeclient.Object) *ruleGroupSyncReconciler {
	fakeClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		WithScheme(testScheme).
		Build()

	return &ruleGroupSyncReconciler{
		seedClient:   fakeClient,
		log:          zap.NewNop().Sugar(),
		recorder:     record.NewFakeRecorder(10),
		versions:     kubermatic.NewFakeVersions(edition.CommunityEdition),
		mlaNamespace: "mla",
	}
}

func TestRuleGroupSync(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		namespacedName types.NamespacedName
		objects        []ctrlruntimeclient.Object
		isSynced       bool
	}{
		{
			name: "sync rulegroup to user cluster namespace",
			namespacedName: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "mla",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
				generateMLARuleGroup("test-rule", "mla", kubermaticv1.RuleGroupTypeMetrics, false),
			},
			isSynced: true,
		},
		{
			name: "do not sync rulegroup to user cluster namespace which has mla disabled",
			namespacedName: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "mla",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, false, false),
				generateMLARuleGroup("test-rule", "mla", kubermaticv1.RuleGroupTypeMetrics, false),
			},
			isSynced: false,
		},
		{
			name: "cleanup rulegroup on user cluster namespace when rulegroup in mla namespace is deleted",
			namespacedName: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "mla",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
				generateMLARuleGroup("test-rule", "cluster-test", kubermaticv1.RuleGroupTypeMetrics, false),
				generateMLARuleGroup("test-rule", "mla", kubermaticv1.RuleGroupTypeMetrics, true),
			},
			isSynced: false,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler := newTestRuleGroupSyncReconciler(tc.objects)

			req := reconcile.Request{NamespacedName: tc.namespacedName}
			_, err := reconciler.Reconcile(ctx, req)
			if err != nil {
				t.Fatalf("Failed to reconcile: %v", err)
			}

			ruleGroup := &kubermaticv1.RuleGroup{}
			err = reconciler.seedClient.Get(ctx, types.NamespacedName{
				Name:      tc.namespacedName.Name,
				Namespace: "cluster-test",
			}, ruleGroup)

			if tc.isSynced {
				if err != nil {
					t.Fatalf("Failed to get synced RuleGroup: %v", err)
				}
			} else {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("Expected not to find RuleGroup anymore, but did: %v", ruleGroup)
				}
			}
		})
	}
}

func generateMLARuleGroup(name, namespace string, ruleGroupType kubermaticv1.RuleGroupType, deleted bool) *kubermaticv1.RuleGroup {
	group := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "RuleGroup",
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: ruleGroupType,
			Data:          generator.GenerateTestRuleGroupData(name),
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		group.DeletionTimestamp = &deleteTime
	}
	return group
}
