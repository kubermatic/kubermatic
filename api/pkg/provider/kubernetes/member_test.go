package kubernetes_test

import (
	"fmt"
	"testing"

	"github.com/go-test/deep"

	kubermaticfakeclentset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/fake"
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
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
	impersonationClient, _, bindingLister, err := createFakeClients(kubermaticObjects)
	if err != nil {
		t.Fatalf("unable to create fake clients, err = %v", err)
	}
	authenticatedUser := createAuthenitactedUser()
	existingProject := createProject("abcd")
	memberEmail := ""
	groupName := fmt.Sprintf("owners-%s", existingProject.Name)

	// act
	target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister)
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
		expectedBindings  []*kubermaticv1.UserProjectBinding
	}{
		{
			name:              "scenario 1: list bindings for the given project",
			authenticatedUser: createAuthenitactedUser(),
			projectToSync:     createProject("1234"),
			existingBindings: []*kubermaticv1.UserProjectBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcdBinding",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						ProjectID: createProject("1234").Name,
						UserEmail: "bob@acme.com",
						Group:     fmt.Sprintf("owners-%s", createProject("123").Name),
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cdBinding",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						ProjectID: createProject("1234").Name,
						UserEmail: "bob@acme.com",
						Group:     fmt.Sprintf("owners-%s", createProject("123").Name),
					},
				},

				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "differentProjectBinding",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						ProjectID: createProject("abcd").Name,
						UserEmail: "bob@acme.com",
						Group:     fmt.Sprintf("owners-%s", createProject("abcd").Name),
					},
				},
			},
			expectedBindings: []*kubermaticv1.UserProjectBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "abcdBinding",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						ProjectID: createProject("1234").Name,
						UserEmail: "bob@acme.com",
						Group:     fmt.Sprintf("owners-%s", createProject("123").Name),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cdBinding",
					},
					Spec: kubermaticv1.UserProjectBindingSpec{
						ProjectID: createProject("1234").Name,
						UserEmail: "bob@acme.com",
						Group:     fmt.Sprintf("owners-%s", createProject("123").Name),
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			kubermaticObjects := []runtime.Object{}
			for _, binding := range tc.existingBindings {
				kubermaticObjects = append(kubermaticObjects, binding)
			}

			impersonationClient, _, bindingLister, err := createFakeClients(kubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			// act
			target := kubernetes.NewProjectMemberProvider(impersonationClient.CreateFakeImpersonatedClientSet, bindingLister)
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

func createFakeClients(kubermaticObjects []runtime.Object) (*FakeImpersonationClient, *kubermaticfakeclentset.Clientset, kubermaticv1lister.UserProjectBindingLister, error) {
	kubermaticClient := kubermaticfakeclentset.NewSimpleClientset(kubermaticObjects...)

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, obj := range kubermaticObjects {
		err := indexer.Add(obj)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	lister := kubermaticv1lister.NewUserProjectBindingLister(indexer)

	return &FakeImpersonationClient{kubermaticClient}, kubermaticClient, lister, nil
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

func createProject(name string) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "kubermatic.io/v1",
					Kind:       "User",
					UID:        "",
					Name:       "my-first-project",
				},
			},
		},
		Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
	}
}
