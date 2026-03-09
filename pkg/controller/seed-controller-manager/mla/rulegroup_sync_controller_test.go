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

package mla

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	mlaNamespace = "mla"
)

func newTestRuleGroupSyncReconciler(objects []ctrlruntimeclient.Object) *ruleGroupSyncReconciler {
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	controller := newRuleGroupSyncController(fakeClient, kubermaticlog.Logger, mlaNamespace)
	reconciler := ruleGroupSyncReconciler{
		Client:                  fakeClient,
		log:                     kubermaticlog.Logger,
		recorder:                events.NewFakeRecorder(10),
		ruleGroupSyncController: controller,
	}
	return &reconciler
}

func TestReconcile(t *testing.T) {
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
				Namespace: mlaNamespace,
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
				generateMLARuleGroup("test-rule", mlaNamespace, kubermaticv1.RuleGroupTypeMetrics, false),
			},
			isSynced: true,
		},
		{
			name: "do not sync rulegroup to user cluster namespace which has mla disabled",
			namespacedName: types.NamespacedName{
				Name:      "test-rule",
				Namespace: mlaNamespace,
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, false, false),
				generateMLARuleGroup("test-rule", mlaNamespace, kubermaticv1.RuleGroupTypeMetrics, false),
			},
			isSynced: false,
		},
		{
			name: "cleanup rulegroup on user cluster namespace when rulegroup in mla namespace is deleted",
			namespacedName: types.NamespacedName{
				Name:      "test-rule",
				Namespace: mlaNamespace,
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
				generateMLARuleGroup("test-rule", "cluster-test", kubermaticv1.RuleGroupTypeMetrics, false),
				generateMLARuleGroup("test-rule", mlaNamespace, kubermaticv1.RuleGroupTypeMetrics, true),
			},
			isSynced: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			reconciler := newTestRuleGroupSyncReconciler(tc.objects)
			req := reconcile.Request{NamespacedName: tc.namespacedName}
			_, err := reconciler.Reconcile(ctx, req)
			assert.NoError(t, err)
			ruleGroup := &kubermaticv1.RuleGroup{}
			err = reconciler.Get(ctx, types.NamespacedName{
				Name:      tc.namespacedName.Name,
				Namespace: "cluster-test",
			}, ruleGroup)
			if tc.isSynced {
				assert.NoError(t, err)
			} else {
				assert.True(t, apierrors.IsNotFound(err), err.Error())
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
			Kind:       kubermaticv1.RuleGroupKindName,
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
		group.Finalizers = []string{"dummy"}
	}
	return group
}
