package rolebinding

import (
	"context"
	"fmt"
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
		expectedRoles             []rbacv1.ClusterRoleBinding
	}{
		{
			name: "scenario 1: test reconcile method for crating cluster role binding",

			expectedRoleNumber:        3,
			expectedRoleBindingNumber: 3,
			expectedRoles: []rbacv1.ClusterRoleBinding{
				*rbacusercluster.GenerateRBACClusterRoleBinding(rbac.OwnerGroupNamePrefix),
				*rbacusercluster.GenerateRBACClusterRoleBinding(rbac.EditorGroupNamePrefix),
				*rbacusercluster.GenerateRBACClusterRoleBinding(rbac.ViewerGroupNamePrefix),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			fakeClient := fake.NewFakeClient()
			r := reconciler{client: fakeClient, ctx: context.TODO()}

			err := createRoles(r.ctx, fakeClient)
			if err != nil {
				t.Fatalf("creteing roles failed: %v", err)
			}
			// create scenario
			err = r.Reconcile()
			if err != nil {
				t.Fatalf("reconcile method error: %v", err)
			}

			roleBindings := &rbacv1.ClusterRoleBindingList{}
			err = r.client.List(r.ctx, &controllerclient.ListOptions{Raw: &metav1.ListOptions{
				TypeMeta: metav1.TypeMeta{
					APIVersion: rbacAPIVersion,
					Kind:       "ClusterRoleBinding",
				},
			},
			}, roleBindings)
			if err != nil {
				t.Fatalf("can't find roleBindings %v", err)
			}

			if len(roleBindings.Items) != test.expectedRoleNumber {
				t.Fatalf("incorrect number of roleBindings were returned, got: %d, want: %d, ", len(roleBindings.Items), test.expectedRoleNumber)
			}

			if !equality.Semantic.DeepEqual(roleBindings.Items, test.expectedRoles) {
				t.Fatalf("incorrect roleBindings were returned, got: %v, want: %v", roleBindings.Items, test.expectedRoles)
			}

		})
	}

}

func TestReconcileUpdateRole(t *testing.T) {

	tests := []struct {
		name                string
		roleName            string
		updatedRoleBinding  *rbacv1.ClusterRoleBinding
		expectedRoleBinding rbacv1.ClusterRoleBinding
	}{
		{
			name:     "scenario 1: test reconcile method when cluster role binding kubermatic:editors was changed",
			roleName: "system:kubermatic:editors",

			updatedRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "system:kubermatic:editors",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "test",
					Kind:     "test",
					Name:     "kubermatic:viewers",
				},
			},
			expectedRoleBinding: *rbacusercluster.GenerateRBACClusterRoleBinding(rbac.EditorGroupNamePrefix),
		},
		{
			name:     "scenario 2: test reconcile method when cluster role kubermatic:viewers was changed",
			roleName: "system:kubermatic:viewers",
			updatedRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "system:kubermatic:viewers",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "test",
					Kind:     "ClusterRole",
					Name:     "kubermatic:viewers",
				},
			},
			expectedRoleBinding: *rbacusercluster.GenerateRBACClusterRoleBinding(rbac.ViewerGroupNamePrefix),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := reconciler{client: fake.NewFakeClient(), ctx: context.TODO()}

			err := createRoles(r.ctx, r.client)
			if err != nil {
				t.Fatalf("creteing roles failed: %v", err)
			}

			if err := r.client.Create(r.ctx, test.updatedRoleBinding); err != nil {
				t.Fatalf("reconcile method error: %v", err)
			}

			// check for updates
			err = r.Reconcile()
			if err != nil {
				t.Fatalf("reconcile method error: %v", err)
			}

			roleBinding := &rbacv1.ClusterRoleBinding{}
			if err := r.client.Get(r.ctx, controllerclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: test.roleName}, roleBinding); err != nil {

				t.Fatalf("can't find cluster roleBinding %v", err)
			}

			// compare roles
			if !equality.Semantic.DeepEqual(roleBinding.RoleRef, test.expectedRoleBinding.RoleRef) {
				t.Fatalf("incorrect cluster roleBinding RoleRef were returned, got: %v, want: %v", roleBinding.RoleRef, test.expectedRoleBinding.RoleRef)
			}

		})
	}

}

func createRoles(ctx context.Context, client controllerclient.Client) error {
	for _, groupName := range rbac.AllGroupsPrefixes {
		defaultClusterRole, err := rbacusercluster.GenerateRBACClusterRole(groupName)
		if err != nil {
			return fmt.Errorf("failed to generate the RBAC Cluster Role: %v", err)
		}
		if err := client.Create(ctx, defaultClusterRole); err != nil {
			return fmt.Errorf("failed to create the RBAC Cluster Role: %v", err)
		}
	}
	return nil
}
