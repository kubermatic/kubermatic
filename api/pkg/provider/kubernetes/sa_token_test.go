package kubernetes_test

import (
	"testing"

	"gopkg.in/square/go-jose.v2/jwt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesclient "k8s.io/client-go/kubernetes"
	fakerestclient "k8s.io/client-go/kubernetes/fake"
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
	}{
		{
			name:          "scenario 1, create token",
			userInfo:      &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saToSync:      createSA("test-1", "my-first-project-ID", "viewers", "1"),
			projectToSync: "my-first-project-ID",
			tokenName:     "test-token",
			expectedSecret: func() *v1.Secret {
				secret := &v1.Secret{}
				secret.Type = "Opaque"
				secret.Namespace = "kubermatic"
				secret.Data = map[string][]byte{}
				secret.Data["token"] = []byte("fake.token")
				secret.Labels = map[string]string{
					kubermaticv1.ProjectIDLabelKey: "my-first-project-ID",
					"name":                         "test-token",
				}
				secret.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: kubermaticv1.SchemeGroupVersion.String(),
						Kind:       kubermaticv1.UserKindName,
						UID:        "",
						Name:       "serviceaccount-1",
					},
				}

				return secret
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, _, err := createFakeKubernetesClients([]runtime.Object{})
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			tokenGenerator := &fakeJWTTokenGenerator{}
			// act
			target := kubernetes.NewServiceAccountTokenProvider(impersonationClient.CreateKubernetesFakeImpersonatedClientSet, tokenGenerator)
			if err != nil {
				t.Fatal(err)
			}

			secret, err := target.Create(tc.userInfo, tc.saToSync, tc.tokenName, tc.projectToSync)
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

// GenerateToken generates new fake token
func (j *fakeJWTTokenGenerator) GenerateToken(claims *jwt.Claims, privateClaims *serviceaccount.TokenClaim) (string, error) {
	return "fake.token", nil
}
