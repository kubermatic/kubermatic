package kubernetes_test

import (
	"context"
	"fmt"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	projectName = "etestxxxq8"
)

func TestGetResourceQuota(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name            string
		projectName     string
		userInfo        *provider.UserInfo
		existingObjects []ctrlruntimeclient.Object
		expectedError   string
	}{
		{
			name:        "scenario 1: get existing resource quota",
			projectName: projectName,
			userInfo:    &provider.UserInfo{Email: "john@acme.com"},
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("project-%s", projectName),
						Namespace: kubermaticv1.ResourceQuotaNamespace,
					},
					Spec: kubermaticv1.ResourceQuotaSpec{
						Subject: kubermaticv1.Subject{
							Name: projectName,
							Kind: "project",
						},
					},
				},
			},
		},
		{
			name:          "scenario 2: get non existing resource quota",
			projectName:   projectName,
			userInfo:      &provider.UserInfo{Email: "john@acme.com"},
			expectedError: fmt.Sprintf("resourcequotas.kubermatic.k8c.io \"project-%s\" not found", projectName),
		},
		{
			name:          "scenario 3: missing user info",
			projectName:   projectName,
			userInfo:      nil,
			expectedError: "a user is missing but required",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}

			targetProvider := kubernetes.NewResourceQuotaProvider(fakeImpersonationClient, fakeClient)

			rq, err := targetProvider.Get(context.Background(), tc.userInfo, tc.projectName, "project")

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error: %s", tc.expectedError)
				}
				if tc.expectedError != err.Error() {
					t.Fatalf("expected error: %s got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if rq.Name != fmt.Sprintf("project-%s", tc.projectName) {
					t.Fatalf("name does not match")
				}
			}
		})
	}
}
