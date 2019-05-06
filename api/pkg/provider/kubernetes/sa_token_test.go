package kubernetes_test

import (
	"strings"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	listers "k8s.io/client-go/listers/core/v1"
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
				secret := genSecret("my-first-project-ID", "serviceaccount-1", "test-token", "1")
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
			projectToSync: genDefaultProject(),
			secrets: []*v1.Secret{
				genSecret("my-first-project-ID", "1", "test-token-1", "1"),
				genSecret("my-first-project-ID", "1", "test-token-2", "2"),
				genSecret("my-first-project-ID", "1", "test-token-3", "3"),
				genSecret("test-ID", "5", "test-token-1", "4"),
				genSecret("project-ID", "6", "test-token-1", "5"),
			},
			expectedTokens: []*v1.Secret{
				rmTokenPrefix(genSecret("my-first-project-ID", "1", "test-token-1", "1")),
				rmTokenPrefix(genSecret("my-first-project-ID", "1", "test-token-2", "2")),
				rmTokenPrefix(genSecret("my-first-project-ID", "1", "test-token-3", "3")),
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
			projectToSync: genDefaultProject(),
			secrets: []*v1.Secret{
				genSecret("my-first-project-ID", "1", "test-token-1", "1"),
				genSecret("my-first-project-ID", "1", "test-token-2", "2"),
				genSecret("my-first-project-ID", "1", "test-token-3", "3"),
				genSecret("test-ID", "5", "test-token-1", "4"),
				genSecret("project-ID", "6", "test-token-1", "5"),
			},
			expectedTokens: []*v1.Secret{
				rmTokenPrefix(genSecret("my-first-project-ID", "1", "test-token-3", "3")),
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

			resultList, err := target.List(tc.userInfo, tc.projectToSync, tc.saToSync, &provider.ServiceAccountTokenListOptions{TokenID: tc.tokenName})
			if err != nil {
				t.Fatal(err)
			}

			if len(resultList) != len(tc.expectedTokens) {
				t.Fatalf("expected equal lengths got %d expected %d", len(resultList), len(tc.expectedTokens))
			}

			sortTokenByName(resultList)
			sortTokenByName(tc.expectedTokens)
			if !equality.Semantic.DeepEqual(resultList, tc.expectedTokens) {
				t.Fatalf("expected  %v got %v", tc.expectedTokens, resultList)
			}
		})
	}
}

func TestGetToken(t *testing.T) {
	// test data
	testcases := []struct {
		name          string
		userInfo      *provider.UserInfo
		saToSync      *kubermaticv1.User
		projectToSync *kubermaticv1.Project
		secrets       []*v1.Secret
		expectedToken *v1.Secret
		tokenToGet    string
	}{
		{
			name:     "scenario 1, get token for the service account 'serviceaccount-1' in project: 'my-first-project-ID'",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync: func() *kubermaticv1.User {
				sa := createSA("test-1", "my-first-project-ID", "viewers", "1")
				// "serviceaccount-" prefix is removed by the provider
				sa.Name = "1"
				return sa
			}(),
			projectToSync: genDefaultProject(),
			secrets: []*v1.Secret{
				genSecret("my-first-project-ID", "1", "test-token-1", "1"),
				genSecret("my-first-project-ID", "1", "test-token-2", "2"),
				genSecret("my-first-project-ID", "1", "test-token-3", "3"),
				genSecret("test-ID", "5", "test-token-1", "4"),
				genSecret("project-ID", "6", "test-token-1", "5"),
			},
			tokenToGet:    "sa-token-3",
			expectedToken: rmTokenPrefix(genSecret("my-first-project-ID", "1", "test-token-3", "3")),
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

			result, err := target.Get(tc.userInfo, tc.tokenToGet)
			if err != nil {
				t.Fatal(err)
			}

			if !equality.Semantic.DeepEqual(result, tc.expectedToken) {
				t.Fatalf("expected  %v got %v", tc.expectedToken, result)
			}
		})
	}
}

func TestUpdateToken(t *testing.T) {
	// test data
	testcases := []struct {
		name          string
		userInfo      *provider.UserInfo
		saToSync      *kubermaticv1.User
		projectToSync *kubermaticv1.Project
		secrets       []*v1.Secret
		expectedToken *v1.Secret
		tokenToUpdate string
		tokenNewName  string
	}{
		{
			name:     "scenario 1, update token name",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync: func() *kubermaticv1.User {
				sa := createSA("test-1", "my-first-project-ID", "viewers", "1")
				// "serviceaccount-" prefix is removed by the provider
				sa.Name = "1"
				return sa
			}(),
			projectToSync: genDefaultProject(),
			secrets: []*v1.Secret{
				genSecret("my-first-project-ID", "1", "test-token-1", "1"),
				genSecret("my-first-project-ID", "1", "test-token-2", "2"),
				genSecret("my-first-project-ID", "1", "test-token-3", "3"),
				genSecret("test-ID", "5", "test-token-1", "4"),
				genSecret("project-ID", "6", "test-token-1", "5"),
			},
			tokenToUpdate: "sa-token-3",
			tokenNewName:  "new-updated-name",
			expectedToken: rmTokenPrefix(genSecret("my-first-project-ID", "1", "new-updated-name", "3")),
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

			result, err := target.Get(tc.userInfo, tc.tokenToUpdate)
			if err != nil {
				t.Fatal(err)
			}
			result.Labels["name"] = tc.tokenNewName
			updated, err := target.Update(tc.userInfo, result)
			if err != nil {
				t.Fatal(err)
			}

			if !equality.Semantic.DeepEqual(updated, tc.expectedToken) {
				t.Fatalf("expected  %v got %v", tc.expectedToken, updated)
			}
		})
	}
}

func TestDeleteToken(t *testing.T) {
	// test data
	testcases := []struct {
		name          string
		userInfo      *provider.UserInfo
		saToSync      *kubermaticv1.User
		projectToSync *kubermaticv1.Project
		secrets       []*v1.Secret
		tokenToDelete string
	}{
		{
			name:     "scenario 1, delete token from service account 'serviceaccount-1' in project: 'my-first-project-ID'",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync: func() *kubermaticv1.User {
				sa := createSA("test-1", "my-first-project-ID", "viewers", "1")
				// "serviceaccount-" prefix is removed by the provider
				sa.Name = "1"
				return sa
			}(),
			projectToSync: test.GenDefaultProject(),
			secrets: []*v1.Secret{
				test.GenDefaultSaToken("my-first-project-ID", "1", "test-token-1", "1"),
				test.GenDefaultSaToken("my-first-project-ID", "1", "test-token-2", "2"),
				test.GenDefaultSaToken("my-first-project-ID", "1", "test-token-3", "3"),
				test.GenDefaultSaToken("test-ID", "5", "test-token-1", "4"),
				test.GenDefaultSaToken("project-ID", "6", "test-token-1", "5"),
			},
			tokenToDelete: "sa-token-3",
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

			// check if token exists first
			existingToken, err := target.Get(tc.userInfo, tc.tokenToDelete)
			if err != nil {
				t.Fatal(err)
			}

			// delete token
			if err := target.Delete(tc.userInfo, existingToken.Name); err != nil {
				t.Fatal(err)
			}

			// validate
			_, err = target.Get(tc.userInfo, tc.tokenToDelete)
			if !errors.IsNotFound(err) {
				t.Fatalf("expected not found error")
			}
		})
	}
}

func rmTokenPrefix(token *v1.Secret) *v1.Secret {
	token.Name = strings.TrimPrefix(token.Name, "sa-token-")
	return token
}
