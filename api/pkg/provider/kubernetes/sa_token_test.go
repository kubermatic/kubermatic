package kubernetes_test

import (
	"sort"
	"testing"

	"gopkg.in/square/go-jose.v2/jwt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesclient "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
	listers "k8s.io/client-go/listers/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func TestCreateToken(t *testing.T) {
	// test data
	testcases := []struct {
		name           string
		userInfo       *provider.UserInfo
		saToSync       *kubermaticv1.User
		projectToSync  string
		expectedSecret *v1.Secret
		tokenName      string
		tokenID        string
		saEmail        string
	}{
		{
			name:          "scenario 1, create token",
			userInfo:      &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync:      createSA("test-1", "my-first-project-ID", "viewers", "1"),
			projectToSync: "my-first-project-ID",
			tokenName:     "test-token",
			tokenID:       "sa-token-1",
			saEmail:       "serviceaccount-1@sa.kubermatic.io",
			expectedSecret: func() *v1.Secret {
				secret := test.GenSecret("my-first-project-ID", "serviceaccount-1", "test-token", "1")
				secret.Name = ""
				return secret
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, indexer, err := createFakeKubernetesClients([]runtime.Object{})
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			tokenGenerator := &fakeJWTTokenGenerator{}
			token, err := tokenGenerator.Generate(serviceaccount.Claims(tc.saEmail, tc.projectToSync, tc.tokenID))
			if err != nil {
				t.Fatalf("unable to generate token, err = %v", err)
			}
			tokenLister := listers.NewSecretLister(indexer)

			// act
			target, err := kubernetes.NewServiceAccountTokenProvider(impersonationClient.CreateKubernetesFakeImpersonatedClientSet, tokenLister)
			if err != nil {
				t.Fatal(err)
			}

			secret, err := target.Create(tc.userInfo, tc.saToSync, tc.projectToSync, tc.tokenName, tc.tokenID, token)
			if err != nil {
				t.Fatal(err)
			}
			secret.Name = ""

			if !equality.Semantic.DeepEqual(secret, tc.expectedSecret) {
				t.Fatalf("expected %v got %v", tc.expectedSecret, secret)
			}
		})
	}
}

func TestListTokens(t *testing.T) {
	// test data
	testcases := []struct {
		name           string
		userInfo       *provider.UserInfo
		saToSync       *kubermaticv1.User
		projectToSync  *kubermaticv1.Project
		secrets        []*v1.Secret
		expectedTokens []*v1.Secret
		tokenName      string
	}{
		{
			name:     "scenario 1, get all tokens for the service account 'serviceaccount-1' in project: 'my-first-project-ID'",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync: func() *kubermaticv1.User {
				sa := createSA("test-1", "my-first-project-ID", "viewers", "1")
				// "serviceaccount-" prefix is removed by the provider
				sa.Name = "1"
				return sa
			}(),
			projectToSync: test.GenDefaultProject(),
			secrets: []*v1.Secret{
				test.GenSecret("my-first-project-ID", "1", "test-token-1", "1"),
				test.GenSecret("my-first-project-ID", "1", "test-token-2", "2"),
				test.GenSecret("my-first-project-ID", "1", "test-token-3", "3"),
				test.GenSecret("test-ID", "5", "test-token-1", "4"),
				test.GenSecret("project-ID", "6", "test-token-1", "5"),
			},
			expectedTokens: []*v1.Secret{
				test.GenSecret("my-first-project-ID", "1", "test-token-1", "1"),
				test.GenSecret("my-first-project-ID", "1", "test-token-2", "2"),
				test.GenSecret("my-first-project-ID", "1", "test-token-3", "3"),
			},
		},
		{
			name:     "scenario 2, get token with specific name",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync: func() *kubermaticv1.User {
				sa := createSA("test-1", "my-first-project-ID", "viewers", "1")
				// "serviceaccount-" prefix is removed by the provider
				sa.Name = "1"
				return sa
			}(),
			projectToSync: test.GenDefaultProject(),
			secrets: []*v1.Secret{
				test.GenSecret("my-first-project-ID", "1", "test-token-1", "1"),
				test.GenSecret("my-first-project-ID", "1", "test-token-2", "2"),
				test.GenSecret("my-first-project-ID", "1", "test-token-3", "3"),
				test.GenSecret("test-ID", "5", "test-token-1", "4"),
				test.GenSecret("project-ID", "6", "test-token-1", "5"),
			},
			expectedTokens: []*v1.Secret{
				test.GenSecret("my-first-project-ID", "1", "test-token-3", "3"),
			},
			tokenName: "test-token-3",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			kubeObjects := []runtime.Object{}
			for _, secret := range tc.secrets {
				kubeObjects = append(kubeObjects, secret)
			}

			impersonationClient, _, indexer, err := createFakeKubernetesClients(kubeObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			tokenLister := listers.NewSecretLister(indexer)

			// act
			target, err := kubernetes.NewServiceAccountTokenProvider(impersonationClient.CreateKubernetesFakeImpersonatedClientSet, tokenLister)
			if err != nil {
				t.Fatal(err)
			}

			resultList, err := target.List(tc.userInfo, tc.projectToSync, tc.saToSync, &provider.ServiceAccountTokenListOptions{TokenName: tc.tokenName})
			if err != nil {
				t.Fatal(err)
			}

			if len(resultList) != len(tc.expectedTokens) {
				t.Fatalf("expected equal lengths got %d expected %d", len(resultList), len(tc.expectedTokens))
			}

			sortByName(resultList)
			sortByName(tc.expectedTokens)
			if !equality.Semantic.DeepEqual(resultList, tc.expectedTokens) {
				t.Fatalf("expected  %v got %v", tc.expectedTokens, resultList)
			}
		})
	}
}

// FakeKubernetesImpersonationClient gives kubernetes client set that uses user impersonation
type FakeKubernetesImpersonationClient struct {
	kubernetesClent *fakerestclient.Clientset
}

func (f *FakeKubernetesImpersonationClient) CreateKubernetesFakeImpersonatedClientSet(impCfg restclient.ImpersonationConfig) (kubernetesclient.Interface, error) {
	return f.kubernetesClent, nil
}

func createFakeKubernetesClients(kubermaticObjects []runtime.Object) (*FakeKubernetesImpersonationClient, *fakerestclient.Clientset, cache.Indexer, error) {
	kubermaticClient := fakerestclient.NewSimpleClientset(kubermaticObjects...)

	indexer, err := createIndexer(kubermaticObjects)
	if err != nil {
		return nil, nil, nil, err
	}

	return &FakeKubernetesImpersonationClient{kubermaticClient}, kubermaticClient, indexer, nil
}

type fakeJWTTokenGenerator struct {
}

// Generate generates new fake token
func (j *fakeJWTTokenGenerator) Generate(claims *jwt.Claims, privateClaims *serviceaccount.CustomTokenClaim) (string, error) {
	return test.TestFakeToken, nil
}

func sortByName(tokens []*v1.Secret) {
	sort.SliceStable(tokens, func(i, j int) bool {
		mi, mj := tokens[i], tokens[j]
		return mi.Name < mj.Name
	})
}
