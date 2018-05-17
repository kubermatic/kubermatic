package rbac

import (
	"testing"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestRBACGeneratorSyncProjectOwner(t *testing.T) {
	tests := []struct {
		name             string
		existingProjects []*kubermaticv1.Project
		existingUser     *kubermaticv1.User
		expectedUser     *kubermaticv1.User
		projectToSync    string
	}{
		{
			name:             "scenario 1: make sure, that the owner of the newly created project is set properly.",
			existingProjects: []*kubermaticv1.Project{createProject("thunderball", createUser("James Bond"))},
			projectToSync:    "thunderball",
			existingUser:     createUser("James Bond"),
			expectedUser:     createExpectedOwnerUser("James Bond", "thunderball"),
		},
		{
			name:             "scenario 2: no op when the owner of the project was set.",
			existingProjects: []*kubermaticv1.Project{createProject("thunderball", createUser("James Bond"))},
			projectToSync:    "thunderball",
			existingUser:     createExpectedOwnerUser("James Bond", "thunderball"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, aProject := range test.existingProjects {
				err := projectIndexer.Add(aProject)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, aProject)
			}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			err := userIndexer.Add(test.existingUser)
			if err != nil {
				t.Fatal(err)
			}
			objs = append(objs, test.existingUser)
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)
			userLister := kubermaticv1lister.NewUserLister(userIndexer)

			// act
			target := Controller{}
			target.kubermaticClient = kubermaticFakeClient
			target.projectLister = projectLister
			target.userLister = userLister
			err = target.sync(test.projectToSync)

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
		{Name: projectName, Group: fmt.Sprintf("%s-owners", projectName)},
	}
	return user
}
