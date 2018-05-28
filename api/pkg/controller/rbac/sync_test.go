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
		name                       string
		projectToSync              *kubermaticv1.Project
		expectedClusterRoleBinding *rbacv1.ClusterRoleBinding
		existingClusterRoleBinding *rbacv1.ClusterRoleBinding
	}{
		{
			name:          "scenario 1: desired RBAC Role Bindings for a project resource are created",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedClusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubermatic:project:owners-thunderball",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							Name:       "thunderball",
							UID:        "", // not generated by the test framework
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
					Name:     "kubermatic:project:owners-thunderball",
				},
			},
		},
		{
			name:          "scenario 2: no op when desicred RBAC Role Bindings exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedClusterRoleBinding: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubermatic:project:owners-thunderball",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							Name:       "thunderball",
							UID:        "", // not generated by the test framework
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
					Name:     "kubermatic:project:owners-thunderball",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			clusterRoleBindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingClusterRoleBinding != nil {
				err := clusterRoleBindingIndexer.Add(test.existingClusterRoleBinding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingClusterRoleBinding)
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleBindingLister := rbaclister.NewClusterRoleBindingLister(clusterRoleBindingIndexer)

			// act
			target := Controller{}
			target.rbacClusterRoleBindingLister = clusterRoleBindingLister
			target.kubeClient = fakeKubeClient
			err := target.ensureProjectRBACRoleBinding(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if test.expectedClusterRoleBinding == nil {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
			}
			action := fakeKubeClient.Actions()[0]
			if !action.Matches("create", "clusterrolebindings") {
				t.Fatalf("unexpected action %#v", action)
			}
			createAction, ok := action.(clienttesting.CreateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRoleBinding), test.expectedClusterRoleBinding) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRoleBinding, createAction.GetObject().(*rbacv1.ClusterRoleBinding)))
			}
		})
	}
}
func TestEnsureProjectRBACRole(t *testing.T) {
	tests := []struct {
		name                string
		projectToSync       *kubermaticv1.Project
		expectedClusterRole *rbacv1.ClusterRole
		existingClusterRole *rbacv1.ClusterRole
	}{
		{
			name:          "scenario 1: desired RBAC Roles for a project resource are created",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			expectedClusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubermatic:project:owners-thunderball",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							Name:       "thunderball",
							UID:        "", // not generated by the test framework
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups:     []string{""},
						Resources:     []string{"projects"},
						ResourceNames: []string{"thunderball"},
						Verbs:         []string{"get", "update", "delete"},
					},
				},
			},
		},
		{
			name:          "scenario 2: no op when desicred RBAC Roles exist",
			projectToSync: createProject("thunderball", createUser("James Bond")),
			existingClusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubermatic:project:owners-thunderball",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io",
							Kind:       "Project",
							Name:       "thunderball",
							UID:        "", // not generated by the test framework
						},
					},
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups:     []string{""},
						Resources:     []string{"projects"},
						ResourceNames: []string{"thunderball"},
						Verbs:         []string{"get", "update", "delete"},
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
			if test.existingClusterRole != nil {
				err := clusterRoleIndexer.Add(test.existingClusterRole)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingClusterRole)
			}
			fakeKubeClient := fake.NewSimpleClientset(objs...)
			clusterRoleLister := rbaclister.NewClusterRoleLister(clusterRoleIndexer)

			// act
			target := Controller{}
			target.rbacClusterRoleLister = clusterRoleLister
			target.kubeClient = fakeKubeClient
			err := target.ensureProjectRBACRole(test.projectToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if test.expectedClusterRole == nil {
				if len(fakeKubeClient.Actions()) != 0 {
					t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
				}
				return
			}

			if len(fakeKubeClient.Actions()) != 1 {
				t.Fatalf("unexpected actions %#v", fakeKubeClient.Actions())
			}
			action := fakeKubeClient.Actions()[0]
			if !action.Matches("create", "clusterroles") {
				t.Fatalf("unexpected action %#v", action)
			}
			createAction, ok := action.(clienttesting.CreateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			if !equality.Semantic.DeepEqual(createAction.GetObject().(*rbacv1.ClusterRole), test.expectedClusterRole) {
				t.Fatalf("%v", diff.ObjectDiff(test.expectedClusterRole, createAction.GetObject().(*rbacv1.ClusterRole)))
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
			Kind:       "Project",
			APIVersion: "kubermatic.k8s.io",
		},
		ObjectMeta: metav1.ObjectMeta{
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
			Kind: "User",
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
