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

package kubernetes_test

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testRuleGroupName        = "test-rule-group"
	testRuleGroupClusterName = "test-rule-group"
)

func TestGetRuleGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		existingObjects   []ctrlruntimeclient.Object
		userInfo          *provider.UserInfo
		cluster           *kubermaticv1.Cluster
		expectedRuleGroup *kubermaticv1.RuleGroup
		expectedError     string
	}{
		{
			name: "get ruleGroup",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
			userInfo:          &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:           genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroup: test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
		},
		{
			name:          "ruleGroup is not found",
			userInfo:      &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:       genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedError: "rulegroups.kubermatic.k8s.io \"test-rule-group\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}

			ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, client)

			ruleGroup, err := ruleGroupProvider.Get(tc.userInfo, tc.cluster, testRuleGroupName)
			if len(tc.expectedError) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				tc.expectedRuleGroup.ResourceVersion = ruleGroup.ResourceVersion
				assert.Equal(t, tc.expectedRuleGroup, ruleGroup)
			} else {
				if err == nil {
					t.Fatalf("expected error message")
				}
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func TestListRuleGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		listOptions        *provider.RuleGroupListOptions
		userInfo           *provider.UserInfo
		cluster            *kubermaticv1.Cluster
		expectedRuleGroups []*kubermaticv1.RuleGroup
		expectedError      string
	}{
		{
			name: "list all ruleGroups",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:  genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroups: []*kubermaticv1.RuleGroup{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
		},
		{
			name: "list all ruleGroups with empty list options",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
			listOptions: &provider.RuleGroupListOptions{},
			userInfo:    &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:     genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroups: []*kubermaticv1.RuleGroup{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
		},
		{
			name: "list ruleGroups with metrics type as list options",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-2", testRuleGroupClusterName, "FakeType"),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
			listOptions: &provider.RuleGroupListOptions{RuleGroupType: kubermaticv1.RuleGroupTypeMetrics},
			userInfo:    &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:     genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroups: []*kubermaticv1.RuleGroup{
				test.GenRuleGroup("test-1", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
				test.GenRuleGroup("test-3", testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
		},
		{
			name:     "ruleGroup is not found",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:  genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}

			ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, client)

			ruleGroups, err := ruleGroupProvider.List(tc.userInfo, tc.cluster, tc.listOptions)
			if len(tc.expectedError) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				if len(tc.expectedRuleGroups) != len(ruleGroups) {
					t.Fatalf("expected to get %d ruleGroups, but got %d", len(tc.expectedRuleGroups), len(ruleGroups))
				}
				ruleGroupMap := make(map[string]*kubermaticv1.RuleGroup)
				for _, ruleGroup := range ruleGroups {
					ruleGroup.ResourceVersion = ""
					ruleGroupMap[ruleGroup.Name] = ruleGroup
				}

				for _, expectedRuleGroup := range tc.expectedRuleGroups {
					ruleGroup, ok := ruleGroupMap[expectedRuleGroup.Name]
					if !ok {
						t.Errorf("expected ruleGroup %s is not in resulting ruleGroups", expectedRuleGroup.Name)
					}
					if diff := deep.Equal(ruleGroup, expectedRuleGroup); diff != nil {
						t.Errorf("Got unexpected ruleGroup. Diff to expected: %v", diff)
					}
				}
			} else {
				if err == nil {
					t.Fatalf("expected error message")
				}
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func TestCreateRuleGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		existingObjects   []ctrlruntimeclient.Object
		userInfo          *provider.UserInfo
		cluster           *kubermaticv1.Cluster
		expectedRuleGroup *kubermaticv1.RuleGroup
		expectedError     string
	}{
		{
			name:              "create ruleGroup",
			userInfo:          &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:           genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroup: test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
		},
		{
			name: "create ruleGroup which already exists",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			},
			userInfo:          &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:           genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroup: test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			expectedError:     "rulegroups.kubermatic.k8s.io \"test-rule-group\" already exists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}

			ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, client)

			_, err := ruleGroupProvider.Create(tc.userInfo, tc.expectedRuleGroup)
			if len(tc.expectedError) == 0 {
				if err != nil {
					t.Fatal(err)
				}

				ruleGroup, err := ruleGroupProvider.Get(tc.userInfo, tc.cluster, tc.expectedRuleGroup.Name)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expectedRuleGroup, ruleGroup)
			} else {
				if err == nil {
					t.Fatalf("expected error message")
				}
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func TestUpdateRuleGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		existingObjects   []ctrlruntimeclient.Object
		userInfo          *provider.UserInfo
		cluster           *kubermaticv1.Cluster
		expectedRuleGroup *kubermaticv1.RuleGroup
		expectedError     string
	}{
		{
			name: "update ruleGroup type",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, "FakeType"),
			},
			userInfo:          &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:           genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroup: test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
		},
		{
			name:              "update ruleGroup which doesn't exist",
			userInfo:          &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:           genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			expectedRuleGroup: test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, kubermaticv1.RuleGroupTypeMetrics),
			expectedError:     "rulegroups.kubermatic.k8s.io \"test-rule-group\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}

			ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, client)
			if len(tc.expectedError) == 0 {
				currentRuleGroup, err := ruleGroupProvider.Get(tc.userInfo, tc.cluster, tc.expectedRuleGroup.Name)
				if err != nil {
					t.Fatal(err)
				}
				tc.expectedRuleGroup.ResourceVersion = currentRuleGroup.ResourceVersion
				_, err = ruleGroupProvider.Update(tc.userInfo, tc.expectedRuleGroup)
				if err != nil {
					t.Fatal(err)
				}
				ruleGroup, err := ruleGroupProvider.Get(tc.userInfo, tc.cluster, tc.expectedRuleGroup.Name)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expectedRuleGroup, ruleGroup)
			} else {
				_, err := ruleGroupProvider.Update(tc.userInfo, tc.expectedRuleGroup)
				if err == nil {
					t.Fatalf("expected error message")
				}
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func TestDeleteRuleGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		userInfo        *provider.UserInfo
		cluster         *kubermaticv1.Cluster
		ruleGroupName   string
		expectedError   string
	}{
		{
			name: "delete ruleGroup",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenRuleGroup(testRuleGroupName, testRuleGroupClusterName, "FakeType"),
			},
			userInfo:      &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:       genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			ruleGroupName: testRuleGroupName,
		},
		{
			name:          "delete ruleGroup which doesn't exist",
			userInfo:      &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:       genCluster(testRuleGroupClusterName, "kubernetes", "my-first-project-ID", "test-rule-group", "john@acme.com"),
			ruleGroupName: testRuleGroupName,
			expectedError: "rulegroups.kubermatic.k8s.io \"test-rule-group\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}

			ruleGroupProvider := kubernetes.NewRuleGroupProvider(fakeImpersonationClient, client)
			err := ruleGroupProvider.Delete(tc.userInfo, tc.cluster, tc.ruleGroupName)
			if len(tc.expectedError) == 0 {
				if err != nil {
					t.Fatal(err)
				}
				_, err = ruleGroupProvider.Get(tc.userInfo, tc.cluster, tc.ruleGroupName)
				assert.True(t, errors.IsNotFound(err))
			} else {
				if err == nil {
					t.Fatalf("expected error message")
				}
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}
