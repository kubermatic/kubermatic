package kubernetes_test

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
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
	}{
		{
			name:     "scenario 1, create service account `test` for owners group",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  createProject("abcd"),
			saName:   "test",
			saGroup:  "owners",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createProject("abcd"),
			},
			expectedSA: func() *kubermaticv1.User {
				sa := &kubermaticv1.User{}
				sa.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: kubermaticv1.SchemeGroupVersion.String(),
						Kind:       kubermaticv1.ProjectKindName,
						UID:        "",
						Name:       "abcd",
					},
				}
				sa.Labels = map[string]string{"group": "owners"}
				return sa
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, _, err := createFakeClients(tc.existingKubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			// act
			target := kubernetes.NewServiceAccountProvider(impersonationClient.CreateFakeImpersonatedClientSet)
			if err != nil {
				t.Fatal(err)
			}

			sa, err := target.CreateServiceAccount(tc.userInfo, tc.project, tc.saName, tc.saGroup)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			if !equality.Semantic.DeepEqual(sa.OwnerReferences, tc.expectedSA.OwnerReferences) {
				t.Fatalf("%v", diff.ObjectDiff(sa.OwnerReferences, tc.expectedSA.OwnerReferences))
			}

			if !equality.Semantic.DeepEqual(sa.Labels, tc.expectedSA.Labels) {
				t.Fatalf("%v", diff.ObjectDiff(sa.Labels, tc.expectedSA.Labels))
			}

			if sa.Spec.Name != tc.saName {
				t.Fatalf("expected %s got %s", tc.saName, sa.Spec.Name)
			}

		})
	}

}
