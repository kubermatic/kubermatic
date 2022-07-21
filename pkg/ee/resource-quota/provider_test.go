//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resourcequota_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	resourcequotas "k8c.io/kubermatic/v2/pkg/ee/resource-quota"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	projectName        = "my-first-project-ID"
	anotherProjectName = "my-second-project-ID"
)

func createResourceProviderHelper(existingObjects []ctrlruntimeclient.Object) *resourcequotas.ResourceQuotaProvider {
	fakeClient := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(existingObjects...).
		Build()

	fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
		return fakeClient, nil
	}

	return resourcequotas.NewResourceQuotaProvider(fakeImpersonationClient, fakeClient)
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

func TestProviderGetResourceQuota(t *testing.T) {
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
						Name: fmt.Sprintf("project-%s", projectName),
						Labels: map[string]string{
							kubermaticv1.ResourceQuotaSubjectNameLabelKey: projectName,
							kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
						},
					},
					Spec: kubermaticv1.ResourceQuotaSpec{
						Subject: kubermaticv1.Subject{
							Name: projectName,
							Kind: kubermaticv1.ProjectSubjectKind,
						},
					},
				},
			},
		},
		{
			name:          "scenario 2: get non existing resource quota",
			projectName:   projectName,
			userInfo:      &provider.UserInfo{Email: "john@acme.com"},
			expectedError: fmt.Sprintf("resource quota not found for project \"%s\"", projectName),
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

			rq, err := rqProvider.Get(context.Background(), tc.userInfo, tc.projectName, kubermaticv1.ProjectSubjectKind)

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

func TestProviderListResourceQuotas(t *testing.T) {
	t.Parallel()

	existingResourceQuotas := []ctrlruntimeclient.Object{
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("project-%s-1", projectName),
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-1", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-1", projectName),
					Kind: kubermaticv1.ProjectSubjectKind,
				},
			},
		},
		&kubermaticv1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("project-%s-2", projectName),
				Labels: map[string]string{
					kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
					kubermaticv1.ResourceQuotaSubjectNameLabelKey: fmt.Sprintf("%s-2", projectName),
				},
			},
			Spec: kubermaticv1.ResourceQuotaSpec{
				Subject: kubermaticv1.Subject{
					Name: fmt.Sprintf("%s-2", projectName),
					Kind: kubermaticv1.ProjectSubjectKind,
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
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
			},
			existingObjects:    existingResourceQuotas,
			expectedListLength: 2,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			rqProvider := createResourceProviderHelper(tc.existingObjects)
			rqList, err := rqProvider.ListUnsecured(context.Background(), tc.labels)
			if err != nil {
				t.Fatal(err)
			}
			if len(rqList.Items) != tc.expectedListLength {
				t.Fatalf("name does not match")
			}
		})
	}
}

func TestProviderCreateResourceQuota(t *testing.T) {
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
				Kind: kubermaticv1.ProjectSubjectKind,
			},
			quota:             createNewQuotaHelper(10),
			expectedQuotaName: fmt.Sprintf("project-%s", projectName),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
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

			if rq.Spec.Quota.CPU.Value() != tc.quota.CPU.Value() {
				t.Fatalf("wrong CPU value")
			}
			if rq.Spec.Quota.Memory.Value() != tc.quota.Memory.Value() {
				t.Fatalf("wrong memory quantity")
			}
			if rq.Spec.Quota.Storage.Value() != tc.quota.Storage.Value() {
				t.Fatalf("wrong storage quantity")
			}
		})
	}
}
