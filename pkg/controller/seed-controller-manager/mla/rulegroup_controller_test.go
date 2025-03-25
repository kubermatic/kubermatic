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
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestRuleGroupReconciler(objects []ctrlruntimeclient.Object, handler http.Handler) (*ruleGroupReconciler, *httptest.Server) {
	fakeClient := fake.
		NewClientBuilder().
		WithObjects(objects...).
		Build()
	ts := httptest.NewServer(handler)

	controller := newRuleGroupController(fakeClient, kubermaticlog.Logger, ts.Client(), ts.URL, ts.URL, mlaNamespace)
	reconciler := ruleGroupReconciler{
		Client:              fakeClient,
		log:                 kubermaticlog.Logger,
		recorder:            record.NewFakeRecorder(10),
		ruleGroupController: controller,
	}
	return &reconciler, ts
}

func TestRuleGroupReconcile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		request      types.NamespacedName
		objects      []ctrlruntimeclient.Object
		requests     []request
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
			requests: []request{
				{
					name: "get",
					request: httptest.NewRequest(http.MethodGet,
						fmt.Sprintf("%s%s/%s", MetricsRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
						nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "post",
					request: httptest.NewRequest(http.MethodPost,
						MetricsRuleGroupConfigEndpoint+defaultNamespace,
						bytes.NewBuffer(generator.GenerateTestRuleGroupData("test-rule"))),
					response: &http.Response{StatusCode: http.StatusAccepted},
				},
			},
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
			requests: []request{
				{
					name: "get",
					request: httptest.NewRequest(http.MethodGet,
						fmt.Sprintf("%s%s/%s", LogRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
						nil),
					response: &http.Response{StatusCode: http.StatusNotFound},
				},
				{
					name: "post",
					request: httptest.NewRequest(http.MethodPost,
						LogRuleGroupConfigEndpoint+defaultNamespace,
						bytes.NewBuffer(generator.GenerateTestRuleGroupData("test-rule"))),
					response: &http.Response{StatusCode: http.StatusAccepted},
				},
			},
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
				generateRuleGroup("test-rule", "test", "type", false),
			},
			expectedErr: true,
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
			requests: []request{
				{
					name: "delete",
					request: httptest.NewRequest(http.MethodDelete,
						fmt.Sprintf("%s%s/%s", MetricsRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
						nil),
					response: &http.Response{StatusCode: http.StatusAccepted},
				},
			},
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
			requests: []request{
				{
					name: "delete",
					request: httptest.NewRequest(http.MethodDelete,
						fmt.Sprintf("%s%s/%s", LogRuleGroupConfigEndpoint, defaultNamespace, "test-rule"),
						nil),
					response: &http.Response{StatusCode: http.StatusAccepted},
				},
			},
			hasFinalizer: false,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			ctx := context.Background()
			r, assertExpectation := buildTestServer(t, testcase.requests...)
			reconciler, server := newTestRuleGroupReconciler(testcase.objects, r)
			request := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testcase.request.Name,
					Namespace: testcase.request.Namespace,
				},
			}
			_, err := reconciler.Reconcile(ctx, request)
			assert.Equal(t, testcase.expectedErr, err != nil)
			ruleGroup := &kubermaticv1.RuleGroup{}
			if err := reconciler.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
				t.Fatalf("unable to get ruleGroup: %v", err)
			}
			assert.Equal(t, testcase.hasFinalizer, kubernetes.HasFinalizer(ruleGroup, ruleGroupFinalizer))
			assertExpectation()
			server.Close()
		})
	}
}

func generateRuleGroup(name, clusterName string, ruleGroupType kubermaticv1.RuleGroupType, deleted bool) *kubermaticv1.RuleGroup {
	group := generator.GenRuleGroup(name, clusterName, ruleGroupType, false)
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		group.DeletionTimestamp = &deleteTime
		group.Finalizers = []string{"dummy"}
	}
	return group
}
