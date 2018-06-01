package rbac

import (
	"fmt"
	"testing"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/fake"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestEnsureProjectIsInActivePhase(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		expectedProject *kubermaticv1.Project
	}{
		{
			name:          "scenario 1: a project's phase is set to Active",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := createProject("thunderball", createUser("James Bond"))
				project.Status.Phase = "Active"
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)

			// act
			target := Controller{}
			target.kubermaticClient = kubermaticFakeClient
			err := target.ensureProjectIsInActivePhase(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedProject == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("update", "projects") {
				t.Fatalf("unexpected action %#v", action)
			}
			updateAction, ok := action.(clienttesting.UpdateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.Project), test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedProject, updateAction.GetObject().(*kubermaticv1.Project)))
			}
		})
	}
}

func TestEnsureProjectInitialized(t *testing.T) {
	tests := []struct {
		name            string
		projectToSync   *kubermaticv1.Project
		expectedProject *kubermaticv1.Project
	}{
		{
			name:          "scenario 1: cleanup finializer is added to a project",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedProject: func() *kubermaticv1.Project {
				project := createProject("thunderball", createUser("James Bond"))
				project.Finalizers = []string{"kubermatic.io/controller-manager-rbac-cleanup"}
				return project
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			objs = append(objs, test.expectedProject)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)

			// act
			target := Controller{}
			target.kubermaticClient = kubermaticFakeClient
			err := target.ensureProjectInitialized(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedProject == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("update", "projects") {
				t.Fatalf("unexpected action %#v", action)
			}
			updateAction, ok := action.(clienttesting.UpdateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.Project), test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedProject, updateAction.GetObject().(*kubermaticv1.Project)))
			}
		})
	}
}

func TestEnsureProjectRBACRoleBinding(t *testing.T) {
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
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"create", "create"},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
							Name:     "admins-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:admins-thunderball",
					},
				},
			},
		},

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Role Bindings exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
							Name:     "admins-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:admins-thunderball",
					},
				},
			},
		},

		// scenario 3
		{
			name:            "scenario 3: update when existing binding doesn't match desired ones",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"update"},
			existingClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
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
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
						Name:     "kubermatic:project-thunderball:admins-thunderball",
					},
				},
			},
			expectedClusterRoleBindings: []*rbacv1.ClusterRoleBinding{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
							Name:     "admins-thunderball",
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "kubermatic:project-thunderball:admins-thunderball",
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
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleBindingLister := rbaclister.NewClusterRoleBindingLister(clusterRoleBindingIndexer)

			// act
			target := Controller{}
			target.rbacClusterRoleBindingLister = clusterRoleBindingLister
			target.kubeClient = fakeKubeClient
			err := target.ensureRBACRoleBindingFor(test.projectToSync.Name, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if len(test.expectedClusterRoleBindings) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedClusterRoleBindings) {
				t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
			}
			for index, action := range fakeKubeClient.Actions() {
				if !action.Matches(test.expectedActions[index], "clusterrolebindings") {
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

func TestEnsureProjectRBACRole(t *testing.T) {
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
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"create", "create"},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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

		// scenario 2
		{
			name:          "scenario 2: no op when desicred RBAC Roles exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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

		// scenario 3
		{
			name:            "scenario 3: update when desired are not the same as expected RBAC Roles",
			projectToSync:   createProject("thunderball", createUser("James Bond")),
			expectedActions: []string{"update"},
			existingClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
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

				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
			},
			expectedClusterRoles: []*rbacv1.ClusterRole{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubermatic:project-thunderball:admins-thunderball",
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
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleLister := rbaclister.NewClusterRoleLister(clusterRoleIndexer)

			// act
			target := Controller{}
			target.rbacClusterRoleLister = clusterRoleLister
			target.kubeClient = fakeKubeClient
			err := target.ensureRBACRoleFor(test.projectToSync.Name, kubermaticv1.ProjectResourceName, kubermaticv1.ProjectKindName, test.projectToSync.GetObjectMeta())

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if len(test.expectedClusterRoles) == 0 {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != len(test.expectedClusterRoles) {
				t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
			}

			for index, action := range fakeKubeClient.Actions() {
				if !action.Matches(test.expectedActions[index], "clusterroles") {
					t.Fatalf("unexpected action %#v", action)
				}
				// TODO: figure out why action.(clienttesting.GenericAction) does not work
				createAction, ok := action.(clienttesting.CreateAction)
				if !ok {
					t.Fatalf("unexpected action %#v", action)
				}
				if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRoles[index]) {
					t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoles[index], createAction.GetObject().(*rbacv1.ClusterRole)))
				}
			}
		})
	}
}

func TestEnsureProjectOwner(t *testing.T) {
	tests := []struct {
		name          string
		projectToSync *kubermaticv1.Project
		existingUser  *kubermaticv1.User
		expectedUser  *kubermaticv1.User
	}{
		{
			name:          "scenario 1: make sure, that the owner of the newly created project is set properly.",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingUser:  createUser("James Bond"),
			expectedUser:  createExpectedOwnerUser("James Bond", "thunderball"),
		},
		{
			name:          "scenario 2: no op when the owner of the project was set.",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingUser:  createExpectedOwnerUser("James Bond", "thunderball"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err := userIndexer.Add(test.existingUser)
			if err != nil {
				t.Fatal(err)
			}
			objs = append(objs, test.existingUser)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			userLister := kubermaticv1lister.NewUserLister(userIndexer)

			// act
			target := Controller{}
			target.kubermaticClient = kubermaticFakeClient
			target.userLister = userLister
			err = target.ensureProjectOwner(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if test.expectedUser == nil {
				if len(kubermaticFakeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
				}
				return
			}
			if len(kubermaticFakeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", kubermaticFakeClient.Actions())
			}

			action := kubermaticFakeClient.Actions()[0]
			if !action.Matches("update", "users") {
				t.Fatalf("unexpected action %#v", action)
			}
			updateAction, ok := action.(clienttesting.UpdateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.User), test.expectedUser) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedUser, updateAction.GetObject().(*kubermaticv1.User)))
			}
		})
	}
}

func createProject(name string, owner *kubermaticv1.User) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.ProjectKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID(name) + "ID",
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
					UID:        owner.GetUID(),
					Name:       owner.Name,
				},
			},
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
	}
}

func createUser(name string) *kubermaticv1.User {
	return &kubermaticv1.User{
		TypeMeta: metav1.TypeMeta{
			Kind: kubermaticv1.UserKindName,
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  "",
			Name: name,
		},
		Spec: kubermaticv1.UserSpec{},
	}
}

func createExpectedOwnerUser(userName, projectName string) *kubermaticv1.User {
	user := createUser(userName)
	user.Spec.Projects = []kubermaticv1.ProjectGroup{
		{Name: projectName, Group: fmt.Sprintf("owners-%s", projectName)},
	}
	return user
}
