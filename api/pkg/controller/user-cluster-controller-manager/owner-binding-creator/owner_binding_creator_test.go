package ownerbindingcreator

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

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
		name            string
		clusterRole     *rbacv1.ClusterRole
		requestName     string
		ownerEmail      string
		expectedBinding rbacv1.ClusterRoleBinding
	}{
		{
			name: "cluster role not found, no error",
		},
		{
			name: "role binding created with cluster owner subject",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name:   "admin",
				Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
			}},
			requestName: "admin",
			ownerEmail:  "test@test.com",
			expectedBinding: rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     "admin",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:     rbacv1.UserKind,
						APIGroup: rbacv1.GroupName,
						Name:     "test@test.com",
					},
				},
			},
		},
		{
			name: "role binding created for no admin role",
			clusterRole: &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{
				Name:   "view",
				Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
			}},
			requestName: "view",
			ownerEmail:  "test@test.com",
			expectedBinding: rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: rbacv1.GroupName,
					Kind:     "ClusterRole",
					Name:     "view",
				},
			},
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
				log:        kubermaticlog.Logger,
				client:     client,
				recorder:   record.NewFakeRecorder(10),
				ownerEmail: tc.ownerEmail,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.clusterRole == nil {
				return
			}

			clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
			if err := client.List(context.Background(), clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{cluster.UserClusterComponentKey: cluster.UserClusterBindingComponentValue}); err != nil {
				t.Fatalf("failed to list cluster role bindigs: %v", err)
			}

			if len(clusterRoleBindingList.Items) != 1 {
				t.Fatalf("expected exactly one binding, got %d", len(clusterRoleBindingList.Items))
			}

			existingBinding := clusterRoleBindingList.Items[0]
			existingBinding.Name = tc.expectedBinding.Name

			if diff := deep.Equal(existingBinding, tc.expectedBinding); diff != nil {
				t.Errorf("bindings are not equal, diff: %v", diff)
			}
		})
	}
}
