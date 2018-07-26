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
			expectedActions: []string{"create", "create", "create", "create","create","create"},
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
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: kubermaticv1.SchemeGroupVersion.String(),
								Kind:       kubermaticv1.ProjectKindName,
								Name:       "thunderball",
								UID:        "thunderballID",
							},
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
							Verbs:         []string{"create", "get", "update", "delete"},
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
							Verbs:         []string{"create", "get", "update", "delete"},
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
		//
		// TODO: uncomment this when existing object are migrated to projects
		//       see: https://github.com/kubermatic/kubermatic/issues/1219
		//
		//
		/*{
			name:            "scenario 2 no-op for a cluster that doesn't belong to a project",
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
					Address: &kubermaticv1.ClusterAddress{},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-abcd",
					},
				},
			},
		},*/
		//
		//  END of TODO
		//
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			kubermaticObjs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingProject != nil {
				err := projectIndexer.Add(test.existingProject)
				if err != nil {
					t.Fatal(err)
				}
				kubermaticObjs = append(kubermaticObjs, test.existingProject)
			}
			//projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)

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
