package rbac

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/fake"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestEnsureDependantsRBACRole(t *testing.T) {
	tests := []struct {
		name                        string
		dependantToSync             *projectResourceQueueItem
		existingProject             *kubermaticv1.Project
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
			expectedActions: []string{"create", "create", "create", "create", "create", "create"},
			existingProject: createProject("thunderball", createUser("James Bond")),

			dependantToSync: &projectResourceQueueItem{
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
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRoleBinding{
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

				&rbacv1.ClusterRoleBinding{
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

				&rbacv1.ClusterRoleBinding{
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
			existingProject: createProject("thunderball", createUser("James Bond")),

			dependantToSync: &projectResourceQueueItem{
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
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRoleBinding{
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

				&rbacv1.ClusterRoleBinding{
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

				&rbacv1.ClusterRoleBinding{
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
			existingProject: createProject("thunderball", createUser("James Bond")),

			dependantToSync: &projectResourceQueueItem{
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
				&rbacv1.ClusterRole{
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
				&rbacv1.ClusterRoleBinding{
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
			name:            "scenario 4 an error is returned when syncing a cluster that doesn't belong to a project",
			expectError:     true,
			existingProject: createProject("thunderball", createUser("James Bond")),
			dependantToSync: &projectResourceQueueItem{
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
			clusterRoleLister := rbaclister.NewClusterRoleLister(clusterRoleIndexer)

			clusterRoleBindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, existingClusterRoleBinding := range test.existingClusterRoleBindings {
				err := clusterRoleIndexer.Add(existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, existingClusterRoleBinding)
			}
			clusterRoleBindingLister := rbaclister.NewClusterRoleBindingLister(clusterRoleBindingIndexer)

			fakeKubeClient := fake.NewSimpleClientset(objs...)
			fakeClusterProvider := &ClusterProvider{
				kubeClient:                   fakeKubeClient,
				rbacClusterRoleBindingLister: clusterRoleBindingLister,
				rbacClusterRoleLister:        clusterRoleLister,
			}

			// act
			target := Controller{}
			test.dependantToSync.clusterProvider = fakeClusterProvider
			err := target.syncProjectResource(test.dependantToSync)

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

			if len(test.expectedClusterRoles) == 0 && len(test.expectedClusterRoleBindings) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedActions) {
				t.Fatalf("unexpected actions expected to get %d, but got %d, actions = %#v", len(test.expectedActions), len(fakeKubeClient.Actions()), fakeKubeClient.Actions())
			}

			allActions := fakeKubeClient.Actions()
			clusterRolesActions := allActions[0:len(test.expectedClusterRoles)]
			offset := 0
			for index, action := range clusterRolesActions {
				offset = offset + 1
				if !action.Matches(test.expectedActions[index], "clusterroles") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createaction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createaction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRoles[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoles[index], createaction.GetObject().(*rbacv1.ClusterRole)))
				}
			}

			clusterRoleBindingActions := allActions[offset:]
			for index, action := range clusterRoleBindingActions {
				if !action.Matches(test.expectedActions[index+offset], "clusterrolebindings") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createAction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBindings[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBindings[index], createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
				}
			}

		})
	}
}
