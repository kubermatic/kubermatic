package kubernetes_test

import (
	"fmt"
	"testing"

	"github.com/go-test/deep"

	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func TestCreateBinding(t *testing.T) {
	// test data
	kubermaticObjects := []runtime.Object{}
	impersonationClient, _, indexer, err := createFakeClients(kubermaticObjects)
	if err != nil {
		t.Fatalf("unable to create fake clients, err = %v", err)
	}
	authenticatedUser := createAuthenitactedUser()
	existingProject := test.GenDefaultProject()
	memberEmail := ""
	groupName := fmt.Sprintf("owners-%s", existingProject.Name)
	bindingLister := kubermaticv1lister.NewUserProjectBindingLister(indexer)
	userLister := kubermaticv1lister.NewUserLister(indexer)
	// act
	target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister, userLister)
	result, err := target.Create(&provider.UserInfo{Email: authenticatedUser.Spec.Email, Group: fmt.Sprintf("owners-%s", existingProject.Name)}, existingProject, memberEmail, groupName)

	// validate
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("received an empty response")
	}
	if result.Spec.Group != groupName {
		t.Fatalf("unexpected result returned, expected %s, got %s", groupName, result.Spec.Group)
	}
	if result.Spec.UserEmail != memberEmail {
		t.Fatalf("unexpected result returned, expected %s, got %s", memberEmail, result.Spec.UserEmail)
	}
	if result.Spec.ProjectID != existingProject.Name {
		t.Fatalf("unexpected result returned, expected %s, got %s", existingProject.Name, result.Spec.ProjectID)
	}
}

func TestListBinding(t *testing.T) {
	// test data
	testcases := []struct {
		name              string
		authenticatedUser *kubermaticv1.User
		projectToSync     *kubermaticv1.Project
		existingBindings  []*kubermaticv1.UserProjectBinding
		existingSA        []*kubermaticv1.User
		expectedBindings  []*kubermaticv1.UserProjectBinding
	}{
		{
			name:              "scenario 1: list bindings for the given project",
			authenticatedUser: createAuthenitactedUser(),
			projectToSync:     test.GenDefaultProject(),
			existingBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("differentProjectBinding", "abcd", "bob@acme.com", "owners"),
			},
			expectedBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
			},
		},
		{
			name:              "scenario 1: filter out service accounts from bindings for the given project",
			authenticatedUser: createAuthenitactedUser(),
			projectToSync:     test.GenDefaultProject(),
			existingSA: []*kubermaticv1.User{
				test.GenServiceAccount("1", "test", "editors", "my-first-project-ID"),
			},
			existingBindings: []*kubermaticv1.UserProjectBinding{

				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				// binding for service account
				createBinding("test", "my-first-project-ID", "serviceaccount-1@localhost", "editors"),
				// binding for regular user with email pattern for service account
				createBinding("fakeServiceAccount", "my-first-project-ID", "serviceaccount-test@localhost", "editors"),
				createBinding("differentProjectBinding", "abcd", "bob@acme.com", "owners"),
			},
			expectedBindings: []*kubermaticv1.UserProjectBinding{
				createBinding("abcdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("cdBinding", "my-first-project-ID", "bob@acme.com", "owners"),
				createBinding("fakeServiceAccount", "my-first-project-ID", "serviceaccount-test@localhost", "editors"),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kubermaticObjects := []runtime.Object{}
			bindingObjects := []runtime.Object{}
			saObjects := []runtime.Object{}
			for _, binding := range tc.existingBindings {
				kubermaticObjects = append(kubermaticObjects, binding)
				bindingObjects = append(bindingObjects, binding)
			}
			for _, sa := range tc.existingSA {
				kubermaticObjects = append(kubermaticObjects, sa)
				saObjects = append(saObjects, sa)
			}
			kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)
			impersonationClient := &FakeImpersonationClient{kubermaticClient}

			bindingIndexer, err := createIndexer(bindingObjects)
			if err != nil {
				t.Fatal(err)
			}
			saIndexer, err := createIndexer(saObjects)
			if err != nil {
				t.Fatal(err)
			}

			bindingLister := kubermaticv1lister.NewUserProjectBindingLister(bindingIndexer)
			userLister := kubermaticv1lister.NewUserLister(saIndexer)
			// act
			target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister, userLister)
			result, err := target.List(&provider.UserInfo{Email: tc.authenticatedUser.Spec.Email, Group: fmt.Sprintf("owners-%s", tc.projectToSync.Name)}, tc.projectToSync, nil)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedBindings) != len(result) {
				t.Fatalf("expected to get %d bindings, but got %d", len(tc.expectedBindings), len(result))
			}
			for _, returnedBinding := range result {
				bindingFound := false
				for _, expectedBinding := range tc.expectedBindings {
					if diff := deep.Equal(returnedBinding, expectedBinding); diff == nil {
						bindingFound = true
						break
					}
				}
				if !bindingFound {
					t.Fatalf("returned binding was not found on the list of expected ones, binding = %#v", returnedBinding)
				}
			}
		})
	}
}

// fakeImpersonationClient gives kubermatic client set that uses user impersonation
type FakeImpersonationClient struct {
	kubermaticClent *kubermaticfakeclentset.Clientset
}

func (f *FakeImpersonationClient) CreateFakeImpersonatedClientSet(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
	return f.kubermaticClent.KubermaticV1(), nil
}

func createFakeClients(kubermaticObjects []runtime.Object) (*FakeImpersonationClient, *kubermaticfakeclentset.Clientset, cache.Indexer, error) {
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)

	indexer, err := createIndexer(kubermaticObjects)
	if err != nil {
		return nil, nil, nil, err
	}

	return &FakeImpersonationClient{kubermaticClient}, kubermaticClient, indexer, nil
}

func createIndexer(kubermaticObjects []runtime.Object) (cache.Indexer, error) {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, obj := range kubermaticObjects {
		err := indexer.Add(obj)
		if err != nil {
			return nil, err
		}
	}
	return indexer, nil
}

func createAuthenitactedUser() *kubermaticv1.User {
	testUserID := "1233"
	testUserName := "user1"
	testUserEmail := "john@acme.com"
	return &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.UserSpec{
			Name:  testUserName,
			ID:    testUserID,
			Email: testUserEmail,
		},
	}

}

func createBinding(name, projectID, email, group string) *kubermaticv1.UserProjectBinding {
	binding := test.GenBinding(projectID, email, group)
	binding.Kind = kubermaticv1.UserProjectBindingKind
	binding.Name = name
	return binding
}
