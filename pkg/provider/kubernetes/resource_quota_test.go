package kubernetes_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	projectName = "etestxxxq8"
)

func createResourceProviderHelper(existingObjects []ctrlruntimeclient.Object) *kubernetes.ResourceQuotaProvider {
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(existingObjects...).
		Build()

	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	return kubernetes.NewResourceQuotaProvider(fakeImpersonationClient, fakeClient)
}

func createNewQuotaHelper(base int) kubermaticv1.ResourceDetails {
	num, _ := resource.ParseQuantity(strconv.Itoa(base))
	asGi, _ := resource.ParseQuantity(fmt.Sprintf("%sGi", strconv.Itoa(base)))
	return kubermaticv1.ResourceDetails{
		CPU:     &num,
		Memory:  &asGi,
		Storage: &asGi,
	}
}

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
			rqProvider := createResourceProviderHelper(tc.existingObjects)

			rq, err := rqProvider.Get(context.Background(), tc.userInfo, tc.projectName, "project")

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

func TestListResourceQuotas(t *testing.T) {
	t.Parallel()

	existingResourceQuotas := []ctrlruntimeclient.Object{
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-%s-1", projectName),
				Namespace: kubermaticv1.ResourceQuotaNamespace,
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: "project",
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-1", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-1", projectName),
					Kind: "project",
				},
			},
		},
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("project-%s-2", projectName),
				Namespace: kubermaticv1.ResourceQuotaNamespace,
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: "project",
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-2", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-2", projectName),
					Kind: "project",
				},
			},
		},
	}

	testcases := []struct {
		name               string
		labels             map[string]string
		existingObjects    []ctrlruntimeclient.Object
		expectedListLength int
	}{
		{
			name:               "scenario 1: listing all existing resource quotas",
			existingObjects:    existingResourceQuotas,
			expectedListLength: len(existingResourceQuotas),
		},
		{
			name: "scenario 2: listing existing resource quotas matching name label",
			labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-1", projectName),
			},
			existingObjects:    existingResourceQuotas,
			expectedListLength: 1,
		},
		{
			name: "scenario 3: listing existing resource quotas matching kind label",
			labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: "project",
			},
			existingObjects:    existingResourceQuotas,
			expectedListLength: 2,
		},
	}
	for _, tc := range testcases {
		rqProvider := createResourceProviderHelper(tc.existingObjects)
		rqList, err := rqProvider.ListUnsecured(context.Background(), tc.labels)
		if err != nil {
			t.Fatal(err)
		}
		if len(rqList.Items) != tc.expectedListLength {
			t.Fatalf("name does not match")
		}
	}
}

func TestCreateResourceQuota(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name              string
		subject           kubermaticv1.Subject
		quota             kubermaticv1.ResourceDetails
		expectedQuotaName string
	}{
		{
			name: "scenario 1: create a new resource quota",
			subject: kubermaticv1.Subject{
				Name: projectName,
				Kind: "project",
			},
			quota:             createNewQuotaHelper(10),
			expectedQuotaName: fmt.Sprintf("project-%s", projectName),
		},
	}

	for _, tc := range testcases {
		rqProvider := createResourceProviderHelper([]ctrlruntimeclient.Object{})
		err := rqProvider.CreateUnsecured(context.Background(), tc.subject, tc.quota)
		if err != nil {
			t.Fatal(err)
		}
		rq, err := rqProvider.GetUnsecured(context.Background(), fmt.Sprintf("%s-%s", tc.subject.Kind, tc.subject.Name))
		if err != nil {
			t.Fatal(err)
		}
		if rq.Name != tc.expectedQuotaName {
			t.Fatalf("expected %s name, got %s", rq.Name, tc.expectedQuotaName)
		}

		labels := rq.GetLabels()
		if labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] != tc.subject.Kind {
			t.Fatalf("missing or wrong kind label")
		}
		if labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] != tc.subject.Name {
			t.Fatalf("mising or wrong name label")
		}

		if rq.Spec.Quota.CPU.Value() != tc.quota.CPU.Value() {
			t.Fatalf("wrong CPU value")
		}
		if rq.Spec.Quota.Memory.Value() != tc.quota.Memory.Value() {
			t.Fatalf("wrong memory quantity")
		}
		if rq.Spec.Quota.Storage.Value() != tc.quota.Storage.Value() {
			t.Fatalf("wrong storage quantity")
		}
	}
}

func TestUpdateResourceQuota(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name          string
		exisingObject ctrlruntimeclient.Object
		newQuota      kubermaticv1.ResourceDetails
	}{
		{
			name: "scenario 1: update existing resource quota",
			exisingObject: &kubermaticv1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("project-%s", projectName),
					Namespace: kubermaticv1.ResourceQuotaNamespace,
				},
				Spec: kubermaticv1.ResourceQuotaSpec{
					Subject: kubermaticv1.Subject{
						Name: projectName,
						Kind: "project",
					},
					Quota: createNewQuotaHelper(10),
				},
			},
			newQuota: createNewQuotaHelper(20),
		},
	}

	for _, tc := range testcases {
		rqProvider := createResourceProviderHelper([]ctrlruntimeclient.Object{tc.exisingObject})

		err := rqProvider.UpdateUnsecured(context.Background(), tc.exisingObject.GetName(), tc.newQuota)
		if err != nil {
			t.Fatal(err)
		}
		rq, err := rqProvider.GetUnsecured(context.Background(), tc.exisingObject.GetName())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(rq.Spec.Quota, tc.newQuota) {
			t.Fatalf("%v", cmp.Diff(rq.Spec.Quota, tc.newQuota))
		}
	}
}
