package role

import (
	"context"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	rbacAPIVersion = "rbac.authorization.k8s.io/v1"
)

func TestReconcileCreate(t *testing.T) {

	tests := []struct {
		name                      string
		expectedRoleNumber        int
		expectedRoleBindingNumber int
		expectedRoles             []rbacv1.ClusterRole
	}{
		{
			name: "scenario 1: test Reconcile method for crating cluster roles",

			expectedRoleNumber:        3,
			expectedRoleBindingNumber: 3,
			expectedRoles: []rbacv1.ClusterRole{
				genTestClusterRole(t, rbac.OwnerGroupNamePrefix),
				genTestClusterRole(t, rbac.EditorGroupNamePrefix),
				genTestClusterRole(t, rbac.ViewerGroupNamePrefix),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			fakeClient := fake.NewFakeClient()
			r := reconciler{client: fakeClient, ctx: context.TODO()}

			// create scenario
			err := r.Reconcile()
			if err != nil {
				t.Fatalf("Reconcile method error: %v", err)
			}

			roles := &rbacv1.ClusterRoleList{}
			err = r.client.List(r.ctx, &controllerclient.ListOptions{Raw: &metav1.ListOptions{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacAPIVersion,
					Kind:       "ClusterRole",
				},
			},
			}, roles)
			if err != nil {
				t.Fatalf("can't find roles %v", err)
			}

			if len(roles.Items) != test.expectedRoleNumber {
				t.Fatalf("incorrect number of roles were returned, got: %d, want: %d, ", len(roles.Items), test.expectedRoleNumber)
			}

			if !equality.Semantic.DeepEqual(roles.Items, test.expectedRoles) {
				t.Fatalf("incorrect roles were returned, got: %v, want: %v", roles.Items, test.expectedRoles)
			}

		})
	}

}

func TestReconcileUpdateRole(t *testing.T) {

	tests := []struct {
		name         string
		roleName     string
		updatedRole  *rbacv1.ClusterRole
		expectedRole rbacv1.ClusterRole
	}{
		{
			name:     "scenario 1: test Reconcile method when cluster role kubermatic:editors was changed",
			roleName: "system:kubermatic:editors",

			updatedRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "system:kubermatic:editors",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"node"},
						Verbs:     []string{"test"},
					},
				},
			},
			expectedRole: genTestClusterRole(t, rbac.EditorGroupNamePrefix),
		},
		{
			name:     "scenario 2: test Reconcile method when cluster role kubermatic:viewers was changed",
			roleName: "system:kubermatic:viewers",
			updatedRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "system:kubermatic:viewers",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"test"},
						Resources: []string{"test"},
						Verbs:     []string{"test"},
					},
				},
			},
			expectedRole: genTestClusterRole(t, rbac.ViewerGroupNamePrefix),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := reconciler{client: fake.NewFakeClient(), ctx: context.TODO()}

			if err := r.client.Create(r.ctx, test.updatedRole); err != nil {
				t.Fatalf("Reconcile method error: %v", err)
			}

			// check for updates
			err := r.Reconcile()
			if err != nil {
				t.Fatalf("Reconcile method error: %v", err)
			}

			role := &rbacv1.ClusterRole{}
			if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: test.roleName}, role); err != nil {

				t.Fatalf("can't find cluster role %v", err)
			}

			// compare roles
			if !equality.Semantic.DeepEqual(role.Rules, test.expectedRole.Rules) {
				t.Fatalf("incorrect cluster role rules were returned, got: %v, want: %v", role.Rules, test.expectedRole.Rules)
			}

			if !equality.Semantic.DeepEqual(role.OwnerReferences, test.expectedRole.OwnerReferences) {
				t.Fatalf("incorrect cluster role OwnerReferences were returned, got: %v, want: %v", role.OwnerReferences, test.expectedRole.OwnerReferences)
			}

		})
	}

}

func genTestClusterRole(t *testing.T, groupName string) rbacv1.ClusterRole {
	role, err := rbacusercluster.GenerateRBACClusterRole(groupName)
	if err != nil {
		t.Fatalf("can't generate role for group %s, error: %v", groupName, err)
	}
	return *role
}
