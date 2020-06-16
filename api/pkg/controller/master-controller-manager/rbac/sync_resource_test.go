package rbac

import (
	"context"
	"reflect"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test"
	fakeInformerProvider "github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test/fake"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeruntime "sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8scorev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestSyncProjectResourcesClusterWide(t *testing.T) {
	tests := []struct {
		name                        string
		dependantToSync             *resourceToProcess
		expectedClusterRoles        []*rbacv1.ClusterRole
		existingClusterRoles        []*rbacv1.ClusterRole
		expectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindings []*rbacv1.ClusterRoleBinding
		expectedActions             []string
		expectError                 bool
	}{
		// scenario 1
		{
			name:            "scenario 1: a proper set of RBAC Role/Binding is generated for a cluster",
			expectedActions: []string{"create", "create", "create", "create", "create", "create", "get", "create", "get", "create", "get", "create", "get", "create", "get", "create", "get", "create"},

			dependantToSync: &resourceToProcess{
				gvr: schema.GroupVersionResource{
					Group:    kubermaticv1.SchemeGroupVersion.Group,
					Version:  kubermaticv1.SchemeGroupVersion.Version,
					Resource: kubermaticv1.ClusterResourceName,
				},
				kind: kubermaticv1.ClusterKindName,
				metaObject: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcd",
						UID:  types.UID("abcdID"),
						Labels: map[string]string{
							kubermaticv1.ProjectIDLabelKey: "thunderball",
						},
					},
					Spec:    kubermaticv1.ClusterSpec{},
					Address: kubermaticv1.ClusterAddress{},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-abcd",
					},
				},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.ClusterResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:owners-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:editors-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:cluster-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ClusterKindName,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:cluster-abcd:viewers-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:            "scenario 2: a proper set of RBAC Role/Binding is generated for an ssh key",
			expectedActions: []string{"create", "create", "create", "create", "create", "create"},

			dependantToSync: &resourceToProcess{
				gvr: schema.GroupVersionResource{
					Group:    kubermaticv1.SchemeGroupVersion.Group,
					Version:  kubermaticv1.SchemeGroupVersion.Version,
					Resource: kubermaticv1.SSHKeyResourceName,
				},
				kind: kubermaticv1.SSHKeyKind,
				metaObject: &kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcd",
						UID:  types.UID("abcdID"),
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID",
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{},
				},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.SSHKeyResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:owners-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:editors-thunderball",
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:usersshkey-abcd:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.SSHKeyKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:usersshkey-abcd:viewers-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: a proper set of RBAC Role/Binding is generated for a userprojectbinding resource",
			expectedActions: []string{"create", "create"},

			dependantToSync: &resourceToProcess{
				gvr: schema.GroupVersionResource{
					Group:    kubermaticv1.SchemeGroupVersion.Group,
					Version:  kubermaticv1.SchemeGroupVersion.Version,
					Resource: kubermaticv1.UserProjectBindingResourceName,
				},
				kind: kubermaticv1.UserProjectBindingKind,
				metaObject: &kubermaticv1.UserProjectBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcd",
						UID:  types.UID("abcdID"),
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID",
							},
						},
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						UserEmail: "bob@acme.com",
						ProjectID: "thunderball",
						Group:     "owners-thunderball",
					},
				},
			},

			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:userprojectbinding-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.UserProjectBindingKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{kubermaticv1.UserProjectBindingResourceName},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
			},

			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:userprojectbinding-abcd:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.UserProjectBindingKind,
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:userprojectbinding-abcd:owners-thunderball",
					},
				},
			},
		},

		// scenario 4
		{
			name:        "scenario 4 an error is returned when syncing a cluster that doesn't belong to a project",
			expectError: true,
			dependantToSync: &resourceToProcess{
				gvr: schema.GroupVersionResource{
					Group:    kubermaticv1.SchemeGroupVersion.Group,
					Version:  kubermaticv1.SchemeGroupVersion.Version,
					Resource: kubermaticv1.ClusterResourceName,
				},
				kind: kubermaticv1.ClusterKindName,
				metaObject: &kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcd",
						UID:  types.UID("abcdID"),
					},
					Spec:    kubermaticv1.ClusterSpec{},
					Address: kubermaticv1.ClusterAddress{},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-abcd",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			clusterRoleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRole := range test.existingClusterRoles {
				err := clusterRoleIndexer.Add(existingClusterRole)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRole)
			}

			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				err := clusterRoleIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRoleBinding)
			}

			fakeMasterClusterClient := fakeruntime.NewFakeClient(objs...)
			// act
			target := resourcesController{
				client:     fakeMasterClusterClient,
				restMapper: getFakeRestMapper(t),
			}
			key := client.ObjectKey{Name: test.dependantToSync.metaObject.GetName(), Namespace: test.dependantToSync.metaObject.GetNamespace()}
			err := target.syncProjectResource(key)

			// validate
			if err != nil && !test.expectError {
				t.Fatal(err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected an error but got nothing")
			}
			if test.expectError {
				return
			}

			{
				var clusterRoleBindings rbacv1.ClusterRoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoleBindings)
				assert.NoError(t, err)

				assert.Len(t, clusterRoleBindings.Items, len(test.existingClusterRoleBindings),
					"cluster contains an different number of ClusterRoleBindings than expected (%d != %d)", len(clusterRoleBindings.Items), len(test.expectedClusterRoleBindings))

			expectedClusterRoleBindingsLoop:
				for _, expectedClusterRoleBinding := range test.existingClusterRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRoleBinding := range clusterRoleBindings.Items {
						if reflect.DeepEqual(*expectedClusterRoleBinding, existingClusterRoleBinding) {
							continue expectedClusterRoleBindingsLoop
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedClusterRoleBinding.Name)
				}
			}

			{
				var clusterRoles rbacv1.ClusterRoleList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoles)
				assert.NoError(t, err)

				assert.Len(t, clusterRoles.Items, len(test.existingClusterRoles),
					"cluster contains an different number of ClusterRoles than expected (%d != %d)", len(clusterRoles.Items), len(test.existingClusterRoles))

			expectedClusterRolesLoop:
				for _, expectedClusterRole := range test.existingClusterRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range clusterRoles.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedClusterRolesLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}
			}
		})
	}
}

func TestSyncProjectResourcesNamespaced(t *testing.T) {
	tests := []struct {
		name                 string
		dependantToSync      *resourceToProcess
		expectedRoles        []*rbacv1.Role
		existingRoles        []*rbacv1.Role
		expectedRoleBindings []*rbacv1.RoleBinding
		existingRoleBindings []*rbacv1.RoleBinding
		expectedActions      []string
		expectError          bool
	}{
		// scenario 1
		{
			name:            "scenario 1: a proper set of RBAC Role/Binding is generated for secrets in kubermatic namespace",
			expectedActions: []string{"create", "create"},

			dependantToSync: &resourceToProcess{
				gvr: schema.GroupVersionResource{
					Group:    k8scorev1.GroupName,
					Version:  k8scorev1.SchemeGroupVersion.Version,
					Resource: "secrets",
				},
				kind: "Secret",
				metaObject: &k8scorev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "abcd",
						Namespace: "kubermatic",
						UID:       types.UID("abcdID"),
						Labels: map[string]string{
							kubermaticv1.ProjectIDLabelKey: "thunderball",
						},
					},
					Type: "Opaque",
					Data: map[string][]byte{
						"token": {0xFF, 0xFF},
					},
				},
			},

			expectedRoles: []*rbacv1.Role{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secret-abcd:owners-thunderball",
						Namespace: "kubermatic",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: k8scorev1.SchemeGroupVersion.String(),
								Kind:       "Secret",
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{k8scorev1.SchemeGroupVersion.Group},
							Resources:     []string{"secrets"},
							ResourceNames: []string{"abcd"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},
			},

			expectedRoleBindings: []*rbacv1.RoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic:secret-abcd:owners-thunderball",
						Namespace: "kubermatic",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: k8scorev1.SchemeGroupVersion.String(),
								Kind:       "Secret",
								Name:       "abcd",
								UID:        "abcdID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "Role",
						Name:     "kubermatic:secret-abcd:owners-thunderball",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			roleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRole := range test.existingRoles {
				err := roleIndexer.Add(existingRole)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingRole)
			}

			roleBindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingRoleBinding := range test.existingRoleBindings {
				err := roleIndexer.Add(existingRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingRoleBinding)
			}

			fakeKubeClient := fake.NewSimpleClientset(objs...)
			// manually set lister as we don't want to start informers in the tests
			fakeInformerFactoryForClusterRole := fakeInformerProvider.NewFakeSharedInformerFactory(fakeKubeClient, metav1.NamespaceAll)
			fakeInformerFactoryForClusterRole.AddFakeRoleBindingInformer(roleBindingIndexer)
			fakeInformerFactoryForClusterRole.AddFakeRoleInformer(roleIndexer)

			fakeMasterClusterClient := fakeruntime.NewFakeClient(objs...)
			// act
			target := resourcesController{
				client:     fakeMasterClusterClient,
				restMapper: getFakeRestMapper(t),
			}
			key := client.ObjectKey{Name: test.dependantToSync.metaObject.GetName(), Namespace: test.dependantToSync.metaObject.GetNamespace()}
			err := target.syncProjectResource(key)

			// validate
			if err != nil && !test.expectError {
				t.Fatal(err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected an error but got nothing")
			}
			if test.expectError {
				return
			}

			if len(test.expectedRoles) == 0 && len(test.expectedRoleBindings) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedActions) {
				t.Fatalf("unexpected actions expected to get %d, but got %d, actions = %#v", len(test.expectedActions), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
			}

			allActions := fakeKubeClient.Actions()
			rolesActions := allActions[0:len(test.expectedRoles)]
			offset := 0
			for index, action := range rolesActions {
				offset++
				if !action.Matches(test.expectedActions[index], "roles") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createaction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createaction.GetObject().(*rbacv1.Role), test.expectedRoles[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedRoles[index], createaction.GetObject().(*rbacv1.Role)))
				}
			}

			roleBindingActions := allActions[offset:]
			for index, action := range roleBindingActions {
				if !action.Matches(test.expectedActions[index+offset], "rolebindings") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createAction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.RoleBinding), test.expectedRoleBindings[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedRoleBindings[index], createAction.GetObject().(*rbacv1.RoleBinding)))
				}
			}

		})
	}
}

func TestEnsureProjectClusterRBACRoleBindingForNamedResource(t *testing.T) {
	tests := []struct {
		name                        string
		projectToSync               *kubermaticv1.Project
		expectedClusterRoleBindings []*rbacv1.ClusterRoleBinding
		existingClusterRoleBindings []*rbacv1.ClusterRoleBinding
		expectedActions             []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Role Bindings for a project resource are created",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Role Bindings exist",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when existing binding doesn't match desired ones",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"update", "update", "update"},
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "owners-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:owners-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "wrong-subject-name",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "editors-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:editors-thunderball",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Subjects: []rbacv1.Subject{
						{
							APIGroup: rbacv1.GroupName,
							Kind:     "Group",
							Name:     "viewers-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:viewers-thunderball",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			clusterRoleBindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				err := clusterRoleBindingIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRoleBinding)
			}
			fakeMasterClusterClient := fakeruntime.NewFakeClient(objs...)

			// act
			err := ensureClusterRBACRoleBindingForNamedResource(fakeMasterClusterClient, test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())
			assert.NoError(t, err)

			{
				var clusterRoleBindings rbacv1.ClusterRoleBindingList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoleBindings)
				assert.NoError(t, err)

				assert.Len(t, clusterRoleBindings.Items, len(test.existingClusterRoleBindings),
					"cluster contains an different number of ClusterRoleBindings than expected (%d != %d)", len(clusterRoleBindings.Items), len(test.expectedClusterRoleBindings))

			expectedClusterRoleBindingsLoop:
				for _, expectedClusterRoleBinding := range test.existingClusterRoleBindings {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRoleBinding := range clusterRoleBindings.Items {
						if reflect.DeepEqual(*expectedClusterRoleBinding, existingClusterRoleBinding) {
							continue expectedClusterRoleBindingsLoop
						}
					}
					t.Fatalf("expected ClusterRoleBinding %q not found in cluster", expectedClusterRoleBinding.Name)
				}
			}
		})
	}
}

func TestEnsureProjectClusterRBACRoleForNamedResource(t *testing.T) {
	tests := []struct {
		name                 string
		projectToSync        *kubermaticv1.Project
		expectedClusterRoles []*rbacv1.ClusterRole
		existingClusterRoles []*rbacv1.ClusterRole
		expectedActions      []string
	}{
		// scenario 1
		{
			name:            "scenario 1: desired RBAC Roles for a project resource are created",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"create", "create", "create"},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Roles exist",
			projectToSync: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when desired are not the same as expected RBAC Roles",
			projectToSync:   test.CreateProject("thunderball", test.CreateUser("James Bond")),
			expectedActions: []string{"update", "update"},
			existingClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:owners-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update", "delete"},
						},
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:viewers-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get"},
						},
					},
				},
			},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:editors-thunderball",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID", // set manually
							},
						},
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups:     []string{kubermaticv1.SchemeGroupVersion.Group},
							Resources:     []string{"projects"},
							ResourceNames: []string{"thunderball"},
							Verbs:         []string{"get", "update"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			clusterRoleIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRole := range test.existingClusterRoles {
				err := clusterRoleIndexer.Add(existingClusterRole)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRole)
			}
			fakeMasterClusterClient := fakeruntime.NewFakeClient(objs...)

			// act
			err := ensureClusterRBACRoleForNamedResource(fakeMasterClusterClient, test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())
			assert.NoError(t, err)

			{
				var clusterRoles rbacv1.ClusterRoleList
				err = fakeMasterClusterClient.List(context.Background(), &clusterRoles)
				assert.NoError(t, err)

				assert.Len(t, clusterRoles.Items, len(test.existingClusterRoles),
					"cluster contains an different number of ClusterRole than expected (%d != %d)", len(clusterRoles.Items), len(test.existingClusterRoles))

			expectedClusterRolesLoop:
				for _, expectedClusterRole := range test.existingClusterRoles {
					// double-iterating over both slices might not be the most efficient way
					// but it spares the trouble of converting pointers to values
					// and then sorting everything for the comparison.

					for _, existingClusterRole := range clusterRoles.Items {
						if reflect.DeepEqual(*expectedClusterRole, existingClusterRole) {
							continue expectedClusterRolesLoop
						}
					}
					t.Fatalf("expected ClusterRole %q not found in cluster", expectedClusterRole.Name)
				}
			}
		})
	}
}
