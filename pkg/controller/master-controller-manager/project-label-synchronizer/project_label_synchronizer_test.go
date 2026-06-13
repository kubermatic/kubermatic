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

package projectlabelsynchronizer

import (
	"context"
	"errors"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconciliation(t *testing.T) {
	const projectName = "label-synchronizer-test"
	testCases := []struct {
		name         string
		masterClient ctrlruntimeclient.Client
		seedClient   ctrlruntimeclient.Client
		// expectedLabels is a map clustername -> LabelMap
		expectedLabels          map[string]map[string]string
		expectedInheritedLabels map[string]map[string]string
	}{
		{
			name:         "Label gets set on matching projectID",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{"foo": "bar"}),
			seedClient: namedClusterWithLabels("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				"foo":                          "bar",
				kubermaticv1.ProjectIDLabelKey: projectName,
			}},
			expectedInheritedLabels: map[string]map[string]string{"baz": {
				"foo": "bar",
			}},
		},
		{
			name:         "Label doesn't get set on wrong projectID",
			masterClient: namedProjectClientWithLabels("wrong", map[string]string{"foo": "bar"}),
			seedClient: namedClusterWithLabels("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: "wrong",
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: "wrong",
			}},
		},
		{
			name:         "Label gets overwritten",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{"foo": "bar"}),
			seedClient: namedClusterWithLabels("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "baz",
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "bar",
			}},
			expectedInheritedLabels: map[string]map[string]string{"baz": {
				"foo": "bar",
			}},
		},
		{
			name:         "No project labels, no update",
			masterClient: namedProjectClientWithLabels(projectName, nil),
			seedClient: namedClusterWithLabels("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
			}},
		},
		{
			name: "Protected labels are not applied",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{
				kubermaticv1.ProjectIDLabelKey:  "not-allowed",
				kubermaticv1.WorkerNameLabelKey: "not-allowed",
			}),
			seedClient: namedClusterWithLabels("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
			}},
		},
		{
			name: "Absent project is handled gracefully",
		},
		{
			name:         "Label removed from project gets removed from cluster",
			masterClient: namedProjectClientWithLabels(projectName, nil),
			seedClient: namedClusterWithLabelsAndInherited("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "bar",
			}, map[string]string{
				"foo": "bar",
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
			}},
			expectedInheritedLabels: map[string]map[string]string{"baz": {}},
		},
		{
			name: "One of several inherited labels is removed from project",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{
				"other": "baz",
			}),
			seedClient: namedClusterWithLabelsAndInherited("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "bar",
				"other":                        "baz",
			}, map[string]string{
				"foo":   "bar",
				"other": "baz",
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
				"other":                        "baz",
			}},
			expectedInheritedLabels: map[string]map[string]string{"baz": {
				"other": "baz",
			}},
		},
		{
			name:         "Manually changed label is not removed, but stops being tracked as inherited",
			masterClient: namedProjectClientWithLabels(projectName, nil),
			seedClient: namedClusterWithLabelsAndInherited("baz", map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "custom-value",
			}, map[string]string{
				"foo": "bar",
			}),
			expectedLabels: map[string]map[string]string{"baz": {
				kubermaticv1.ProjectIDLabelKey: projectName,
				"foo":                          "custom-value",
			}},
			expectedInheritedLabels: map[string]map[string]string{"baz": {}},
		},
		{
			name:         "Cluster belonging to a different project is left untouched",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{"foo": "bar"}),
			seedClient: fake.NewClientBuilder().WithObjects(
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "baz",
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
					},
				},
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "other",
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "some-other-project"},
					},
				},
			).Build(),
			expectedLabels: map[string]map[string]string{
				"baz": {
					"foo":                          "bar",
					kubermaticv1.ProjectIDLabelKey: projectName,
				},
				"other": {
					kubermaticv1.ProjectIDLabelKey: "some-other-project",
				},
			},
			expectedInheritedLabels: map[string]map[string]string{"baz": {
				"foo": "bar",
			}},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.masterClient == nil {
				tc.masterClient = fake.NewClientBuilder().Build()
			}
			if tc.seedClient == nil {
				tc.seedClient = fake.NewClientBuilder().Build()
			}

			ctx := context.Background()
			r := &reconciler{
				log:                     kubermaticlog.Logger,
				masterClient:            tc.masterClient,
				seedClients:             map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
				workerNameLabelSelector: labels.Everything(),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: projectName}}
			_, err := r.Reconcile(ctx, request)
			if err != nil {
				t.Fatalf("Error when reconciling: %v", err)
			}

			clusters := &kubermaticv1.ClusterList{}
			if err := tc.seedClient.List(ctx, clusters); err != nil {
				t.Fatalf("Error listing clusters: %v", err)
			}

			for _, cluster := range clusters.Items {
				if expected := tc.expectedLabels[cluster.Name]; !diff.SemanticallyEqual(expected, cluster.Labels) {
					t.Errorf("Expected labels on cluster %q do not match actual labels:\n%v", cluster.Name, diff.ObjectDiff(expected, cluster.Labels))
				}

				if expected := tc.expectedInheritedLabels[cluster.Name]; !diff.SemanticallyEqual(expected, cluster.Status.InheritedLabels) {
					t.Errorf("Expected inherited labels on cluster %q do not match actual inherited labels:\n%v", cluster.Name, diff.ObjectDiff(expected, cluster.Status.InheritedLabels))
				}
			}
		})
	}
}

// TestReconciliationErrors verifies that failures to patch the cluster's labels
// or its status are surfaced as errors from Reconcile, instead of being swallowed.
func TestReconciliationErrors(t *testing.T) {
	const projectName = "label-synchronizer-test"

	testCases := []struct {
		name         string
		masterClient ctrlruntimeclient.Client
		seedClient   ctrlruntimeclient.Client
	}{
		{
			name:         "Error patching cluster labels is returned",
			masterClient: namedProjectClientWithLabels(projectName, map[string]string{"foo": "bar"}),
			seedClient: fake.NewClientBuilder().
				WithObjects(&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "baz",
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
					},
				}).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(ctx context.Context, c ctrlruntimeclient.WithWatch, obj ctrlruntimeclient.Object, patch ctrlruntimeclient.Patch, opts ...ctrlruntimeclient.PatchOption) error {
						return errors.New("boom")
					},
				}).
				Build(),
		},
		{
			name:         "Error patching cluster status is returned",
			masterClient: namedProjectClientWithLabels(projectName, nil),
			seedClient: fake.NewClientBuilder().
				WithObjects(&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "baz",
						Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectName},
					},
					Status: kubermaticv1.ClusterStatus{
						InheritedLabels: map[string]string{"foo": "bar"},
					},
				}).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourcePatch: func(ctx context.Context, c ctrlruntimeclient.Client, subResourceName string, obj ctrlruntimeclient.Object, patch ctrlruntimeclient.Patch, opts ...ctrlruntimeclient.SubResourcePatchOption) error {
						return errors.New("boom")
					},
				}).
				Build(),
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &reconciler{
				log:                     kubermaticlog.Logger,
				masterClient:            tc.masterClient,
				seedClients:             map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
				workerNameLabelSelector: labels.Everything(),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: projectName}}
			if _, err := r.Reconcile(context.Background(), request); err == nil {
				t.Fatal("expected Reconcile to return an error, got nil")
			}
		})
	}
}

func namedProjectClientWithLabels(name string, labels map[string]string) ctrlruntimeclient.Client {
	return fake.NewClientBuilder().WithObjects(&kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}).Build()
}

func namedClusterWithLabels(name string, labels map[string]string) ctrlruntimeclient.Client {
	return fake.NewClientBuilder().WithObjects(&kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}).Build()
}

func namedClusterWithLabelsAndInherited(name string, labels, inheritedLabels map[string]string) ctrlruntimeclient.Client {
	return fake.NewClientBuilder().WithObjects(&kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: kubermaticv1.ClusterStatus{
			InheritedLabels: inheritedLabels,
		},
	}).Build()
}
