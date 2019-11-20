package projectlabelsynchronizer

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.masterClient == nil {
				tc.masterClient = fakectrlruntimeclient.NewFakeClient()
			}
			if tc.seedClient == nil {
				tc.seedClient = fakectrlruntimeclient.NewFakeClient()
			}
			r := &reconciler{
				log:                     kubermaticlog.Logger,
				masterClient:            tc.masterClient,
				seedClients:             map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
				workerNameLabelSelector: labels.Everything(),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: projectName}}
			_, err := r.Reconcile(request)
			if err != nil {
				t.Fatalf("Error when reconciling: %v", err)
			}

			clusters := &kubermaticv1.ClusterList{}
			if err := tc.seedClient.List(context.Background(), clusters); err != nil {
				t.Fatalf("Error listing clusters: %v", err)
			}

			for _, cluster := range clusters.Items {
				if diff := deep.Equal(cluster.Labels, tc.expectedLabels[cluster.Name]); diff != nil {
					t.Errorf("Expected labels on cluster %q do not match actual labels, diff: %v", cluster.Name, diff)
				}

				if diff := deep.Equal(cluster.InheritedLabels, tc.expectedInheritedLabels[cluster.Name]); diff != nil {
					t.Errorf("Expected inherited labels on cluster %q do not match actual inherited labels, diff: %v", cluster.Name, diff)
				}
			}
		})
	}
}

func namedProjectClientWithLabels(name string, labels map[string]string) ctrlruntimeclient.Client {
	return fakectrlruntimeclient.NewFakeClient(&kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	})
}

func namedClusterWithLabels(name string, labels map[string]string) ctrlruntimeclient.Client {
	return fakectrlruntimeclient.NewFakeClient(&kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	})
}
