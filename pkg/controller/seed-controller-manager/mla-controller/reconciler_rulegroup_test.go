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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/test/generator"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestRuleGroupReconciler(objects []ctrlruntimeclient.Object) (*ruleGroupReconciler, cortex.Client) {
	fakeClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		WithScheme(testScheme).
		Build()

	cClient := cortex.NewFakeClient()

	return &ruleGroupReconciler{
		seedClient:   fakeClient,
		log:          zap.NewNop().Sugar(),
		recorder:     record.NewFakeRecorder(10),
		versions:     kubermatic.NewFakeVersions(edition.CommunityEdition),
		mlaNamespace: "mla",
		cortexClientProvider: func() cortex.Client {
			return cClient
		},
	}, cClient
}

func TestRuleGroupReconcile(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name         string
		request      types.NamespacedName
		objects      []ctrlruntimeclient.Object
		expectedErr  bool
		hasFinalizer bool
	}{
		{
			name: "create metrics rule group",
			request: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "cluster-test",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, false, false),
				generateRuleGroup("test-rule", "test", kubermaticv1.RuleGroupTypeMetrics, false),
			},
			// requests: []request{
			// 	{
			// 		name: "get",
			// 		request: httptest.NewRequest(http.MethodGet,
			// 			fmt.Sprintf("%s%s/%s", MetricsRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
			// 			nil),
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "post",
			// 		request: httptest.NewRequest(http.MethodPost,
			// 			MetricsRuleGroupConfigEndpoint+defaultNamespace,
			// 			bytes.NewBuffer(generator.GenerateTestRuleGroupData("test-rule"))),
			// 		response: &http.Response{StatusCode: http.StatusAccepted},
			// 	},
			// },
			hasFinalizer: true,
		},
		{
			name: "create logs rule group",
			request: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "cluster-test",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", false, true, false),
				generateRuleGroup("test-rule", "test", kubermaticv1.RuleGroupTypeLogs, false),
			},
			// requests: []request{
			// 	{
			// 		name: "get",
			// 		request: httptest.NewRequest(http.MethodGet,
			// 			fmt.Sprintf("%s%s/%s", LogRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
			// 			nil),
			// 		response: &http.Response{StatusCode: http.StatusNotFound},
			// 	},
			// 	{
			// 		name: "post",
			// 		request: httptest.NewRequest(http.MethodPost,
			// 			LogRuleGroupConfigEndpoint+defaultNamespace,
			// 			bytes.NewBuffer(generator.GenerateTestRuleGroupData("test-rule"))),
			// 		response: &http.Response{StatusCode: http.StatusAccepted},
			// 	},
			// },
			hasFinalizer: true,
		},
		{
			name: "create rule group with unknown type",
			request: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "cluster-test",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, true, false),
				generateRuleGroup("test-rule", "test", "nonexisting-type", false),
			},
			expectedErr:  true,
			hasFinalizer: true,
		},
		{
			name: "clean up metrics rule group",
			request: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "cluster-test",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, true, false),
				generateRuleGroup("test-rule", "test", kubermaticv1.RuleGroupTypeMetrics, true),
			},
			// requests: []request{
			// 	{
			// 		name: "delete",
			// 		request: httptest.NewRequest(http.MethodDelete,
			// 			fmt.Sprintf("%s%s/%s", MetricsRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
			// 			nil),
			// 		response: &http.Response{StatusCode: http.StatusAccepted},
			// 	},
			// },
			hasFinalizer: false,
		},
		{
			name: "clean up logs rule group",
			request: types.NamespacedName{
				Name:      "test-rule",
				Namespace: "cluster-test",
			},
			objects: []ctrlruntimeclient.Object{
				generateCluster("test", true, true, false),
				generateRuleGroup("test-rule", "test", kubermaticv1.RuleGroupTypeLogs, true),
			},
			// requests: []request{
			// 	{
			// 		name: "delete",
			// 		request: httptest.NewRequest(http.MethodDelete,
			// 			fmt.Sprintf("%s%s/%s", LogRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
			// 			nil),
			// 		response: &http.Response{StatusCode: http.StatusAccepted},
			// 	},
			// },
			hasFinalizer: false,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, _ := newTestRuleGroupReconciler(tc.objects)
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.request.Name,
					Namespace: tc.request.Namespace,
				},
			}

			_, err := reconciler.Reconcile(ctx, request)
			if tc.expectedErr != (err != nil) {
				t.Fatalf("ExpectedErr = %v, but got: %v", tc.expectedErr, err)
			}

			ruleGroup := &kubermaticv1.RuleGroup{}
			if err := reconciler.seedClient.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
				t.Fatalf("Failed to get ruleGroup: %v", err)
			}

			if tc.hasFinalizer != kubernetes.HasFinalizer(ruleGroup, ruleGroupFinalizer) {
				t.Fatalf("Expected Finalizer=%v, failed to assert that.", tc.hasFinalizer)
			}
		})
	}
}

func generateRuleGroup(name, clusterName string, ruleGroupType kubermaticv1.RuleGroupType, deleted bool) *kubermaticv1.RuleGroup {
	group := generator.GenRuleGroup(name, clusterName, ruleGroupType, false)
	if deleted {
		deleteTime := metav1.Now()
		group.DeletionTimestamp = &deleteTime
	}
	return group
}
