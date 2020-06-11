package clusterrolelabeler

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	"github.com/kubermatic/kubermatic/pkg/handler/v1/cluster"
	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name           string
		clusterRole    *rbacv1.ClusterRole
		requestName    string
		expectedLabels map[string]string
	}{
		{
			name: "cluster role not found, no error",
		},
		{
			name: "label added to view cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "view",
			}},
			requestName:    "view",
			expectedLabels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
		},
		{
			name: "label added to edit cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "edit",
			}},
			requestName:    "edit",
			expectedLabels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
		},
		{
			name: "label added to admin cluster role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name: "admin",
			}},
			requestName:    "admin",
			expectedLabels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var client ctrlruntimeclient.Client
			if tc.clusterRole != nil {
				client = fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.clusterRole)
			} else {
				client = fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme)
			}
			r := &reconciler{
				log:      kubermaticlog.Logger,
				client:   client,
				recorder: record.NewFakeRecorder(10),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.clusterRole == nil {
				return
			}

			clusterRole := &rbacv1.ClusterRole{}
			if err := client.Get(context.Background(), request.NamespacedName, clusterRole); err != nil {
				t.Fatalf("failed to get cluster role: %v", err)
			}

			if diff := deep.Equal(clusterRole.Labels, tc.expectedLabels); diff != nil {
				t.Errorf("cluster role doesn't have expected labels, diff: %v", diff)
			}
		})
	}
}
