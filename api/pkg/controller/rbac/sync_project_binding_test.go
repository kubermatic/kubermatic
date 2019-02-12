package rbac

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/tools/cache"

	kubermaticfakeclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clienttesting "k8s.io/client-go/testing"
)

func TestEnsureNotProjectOwnerForBinding(t *testing.T) {
	tests := []struct {
		name             string
		bindingToSync    *kubermaticv1.UserProjectBinding
		existingProject  *kubermaticv1.Project
		existingUsers    []*kubermaticv1.User
		expectedProject  *kubermaticv1.Project
		existingBindings []*kubermaticv1.UserProjectBinding
	}{
		{
			name:            "scenario 1: the owner reference is removed from a project (no previous owners) for James Bond - an editor",
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
		},
		{
			name: "scenario 2: no - op the owner reference already removed from a project (no previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync: createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
		},
		{
			name: "scenario 3: the owner reference was removed from a project (with previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = append(prj.OwnerReferences, metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        "",
					Name:       "Bob",
				})
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{createUser("James Bond"), createUser("Bob")},
			bindingToSync: createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{
					metav1.OwnerReference{
						APIVersion: kubermaticv1.SchemeGroupVersion.String(),
						Kind:       kubermaticv1.UserKindName,
						UID:        "",
						Name:       "Bob",
					},
				}
				return prj
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, user := range test.existingUsers {
				err := userIndexer.Add(user)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, user)
			}
			bindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, binding := range test.existingBindings {
				err := bindingIndexer.Add(binding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, binding)
			}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingProject != nil {
				err := projectIndexer.Add(test.existingProject)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingProject)
			}
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: kubermaticFakeClient,
			}
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)
			projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)

			// act
			target := Controller{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.userLister = userLister
			target.userProjectBindingLister = bindingLister
			target.projectLister = projectLister
			err := target.ensureNotProjectOwnerForBinding(test.bindingToSync)

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
			updatedProject := updateAction.GetObject().(*kubermaticv1.Project)
			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}

func TestEnsureProjectOwnerForBinding(t *testing.T) {
	tests := []struct {
		name             string
		bindingToSync    *kubermaticv1.UserProjectBinding
		existingProject  *kubermaticv1.Project
		existingUsers    []*kubermaticv1.User
		expectedProject  *kubermaticv1.Project
		existingBindings []*kubermaticv1.UserProjectBinding
	}{
		{
			name:            "scenario 1: no-op the owner reference already attached to the project",
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
		},
		{
			name: "scenario 2: expected owner reference was added to a project - no previous owners)",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: createProject("thunderball", createUser("James Bond")),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond"), createUser("Bob")},
			bindingToSync:   createExpectedOwnerBinding("Bob", createProject("thunderball", createUser("Bob"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = append(prj.OwnerReferences, metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        "",
					Name:       "Bob",
				})
				return prj
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			objs := []runtime.Object{}
			userIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, user := range test.existingUsers {
				err := userIndexer.Add(user)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, user)
			}
			bindingIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			for _, binding := range test.existingBindings {
				err := bindingIndexer.Add(binding)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, binding)
			}
			projectIndexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			if test.existingProject != nil {
				err := projectIndexer.Add(test.existingProject)
				if err != nil {
					t.Fatal(err)
				}
				objs = append(objs, test.existingProject)
			}
			kubermaticFakeClient := kubermaticfakeclientset.NewSimpleClientset(objs...)
			fakeMasterClusterProvider := &ClusterProvider{
				kubermaticClient: kubermaticFakeClient,
			}
			userLister := kubermaticv1lister.NewUserLister(userIndexer)
			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)
			projectLister := kubermaticv1lister.NewProjectLister(projectIndexer)

			// act
			target := Controller{}
			target.masterClusterProvider = fakeMasterClusterProvider
			target.userLister = userLister
			target.userProjectBindingLister = bindingLister
			target.projectLister = projectLister
			err := target.ensureProjectOwnerForBinding(test.bindingToSync)

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
			updatedProject := updateAction.GetObject().(*kubermaticv1.Project)
			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}
