package rolecloner

import (
	"reflect"
	"sort"
	"testing"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var nowTime metav1.Time

func TestReconcile(t *testing.T) {
	nowTime = metav1.Now()
	testCases := []struct {
		name             string
		objects          []runtime.Object
		expectedRoles    []rbacv1.Role
		requestName      string
		requestNamespace string
	}{
		{
			name: "role not found, no error",
		},
		{
			name: "delete role view for all namespaces",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "kube-system",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
			},
			objects: []runtime.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "view",
						Namespace:         "kube-system",
						Finalizers:        []string{kubermaticapiv1.UserClusterRoleCleanupFinalizer},
						Labels:            map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
						DeletionTimestamp: &nowTime,
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
		{
			name: "clone role for all namespaces",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{kubermaticapiv1.UserClusterRoleCleanupFinalizer},
						Labels:     map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
			},
			objects: []runtime.Object{
				&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{
					Name:      "view",
					Namespace: "kube-system",
					Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
				}},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
		{
			name: "update role view for all namespaces",
			expectedRoles: []rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{kubermaticapiv1.UserClusterRoleCleanupFinalizer},
						Labels:     map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
			},
			objects: []runtime.Object{
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "view",
						Namespace:  "kube-system",
						Finalizers: []string{kubermaticapiv1.UserClusterRoleCleanupFinalizer},
						Labels:     map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
					Rules: []rbacv1.PolicyRule{
						{
							Verbs:     []string{"*"},
							APIGroups: []string{"*"},
							Resources: []string{"*"},
						},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "default",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				&rbacv1.Role{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "view",
						Namespace: "test",
						Labels:    map[string]string{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
				},
			},
			requestName:      "view",
			requestNamespace: "kube-system",
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var client ctrlruntimeclient.Client
			if tc.expectedRoles != nil {
				client = fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.objects...)
			} else {
				client = fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme)
			}
			r := &reconciler{
				log:      kubermaticlog.Logger,
				client:   client,
				recorder: record.NewFakeRecorder(10),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName, Namespace: tc.requestNamespace}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.expectedRoles == nil {
				return
			}

			existingRoleList := &rbacv1.RoleList{}
			if err := r.client.List(r.ctx, existingRoleList, ctrlruntimeclient.MatchingLabels{cluster.UserClusterComponentKey: cluster.UserClusterRoleComponentValue}); err != nil {
				t.Fatalf("failed to get role: %v", err)
			}

			existingRoles := existingRoleList.Items

			if len(existingRoles) != len(tc.expectedRoles) {
				t.Fatalf("roles are not equal, expected length %d got %d", len(tc.expectedRoles), len(existingRoles))
			}

			var newExistingRoles []rbacv1.Role
			// get rid of time format differences
			for _, role := range existingRoles {
				role.DeletionTimestamp = nil
				newExistingRoles = append(newExistingRoles, role)
			}
			sortRoles(newExistingRoles)
			sortRoles(tc.expectedRoles)

			if !reflect.DeepEqual(newExistingRoles, tc.expectedRoles) {
				t.Fatalf("roles are not equal, expected %v got %v", tc.expectedRoles, newExistingRoles)
			}

		})
	}
}

func sortRoles(roles []rbacv1.Role) {
	sort.SliceStable(roles, func(i, j int) bool {
		mi, mj := roles[i], roles[j]
		return mi.Name < (mj.Name)
	})
}
