package kubernetes_test

import (
	"testing"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateServiceAccount(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		userInfo                  *provider.UserInfo
		saName                    string
		saGroup                   string
		expectedSA                *kubermaticv1.User
		expectedSAName            string
	}{
		{
			name:     "scenario 1, create service account `test` for editors group",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  test.GenDefaultProject(),
			saName:   "test",
			saGroup:  "editors-my-first-project-ID",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				test.GenDefaultProject(),
			},
			expectedSA:     createSA("test", "my-first-project-ID", "editors", "1"),
			expectedSAName: "serviceaccount-1",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, indexer, err := createFakeClients(tc.existingKubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			saLister := kubermaticv1lister.NewUserLister(indexer)

			// act
			target := kubernetes.NewServiceAccountProvider(impersonationClient.CreateFakeImpersonatedClientSet, saLister, "localhost")
			if err != nil {
				t.Fatal(err)
			}

			sa, err := target.Create(tc.userInfo, tc.project, tc.saName, tc.saGroup)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			//remove autogenerated fields
			sa.Name = tc.expectedSAName
			sa.Spec.Email = ""
			sa.Spec.ID = ""

			if !equality.Semantic.DeepEqual(sa, tc.expectedSA) {
				t.Fatalf("expected %v got %v", tc.expectedSA, sa)
			}
		})
	}
}

func TestList(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		saName                    string
		userInfo                  *provider.UserInfo
		expectedSA                []*kubermaticv1.User
	}{
		{
			name:     "scenario 1, get existing service accounts",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  test.GenDefaultProject(),
			saName:   "test-1",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test-1", "my-first-project-ID", "editors", "1"),
				createSA("test-2", "abcd", "viewers", "2"),
				createSA("test-1", "dcba", "viewers", "3"),
			},
			expectedSA: []*kubermaticv1.User{
				createSA("test-1", "my-first-project-ID", "editors", "1"),
			},
		},
		{
			name:     "scenario 2, service accounts not found for the project",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  test.GenDefaultProject(),
			saName:   "test",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test", "bbbb", "editors", "1"),
				createSA("fake", "abcd", "editors", "2"),
			},
			expectedSA: []*kubermaticv1.User{},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, indexer, err := createFakeClients(tc.existingKubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}
			saLister := kubermaticv1lister.NewUserLister(indexer)
			// act
			target := kubernetes.NewServiceAccountProvider(impersonationClient.CreateFakeImpersonatedClientSet, saLister, "localhost")
			if err != nil {
				t.Fatal(err)
			}

			saList, err := target.List(tc.userInfo, tc.project, &provider.ServiceAccountListOptions{ServiceAccountName: tc.saName})
			// validate
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(saList, tc.expectedSA) {
				t.Fatalf("expected %v got %v", tc.expectedSA, saList)
			}
		})
	}
}

func createSA(name, projectName, group, id string) *kubermaticv1.User {
	sa := test.GenServiceAccount(id, name, group, projectName)
	// remove autogenerated values
	sa.OwnerReferences[0].UID = ""
	sa.Spec.Email = ""
	sa.Spec.ID = ""

	return sa
}
