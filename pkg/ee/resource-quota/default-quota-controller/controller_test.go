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

package defaultcontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                   string
		expectedResourceQuotas []kubermaticv1.ResourceQuota
		masterClient           ctrlruntimeclient.Client
		seedClients            map[string]ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: create default project quota",
			expectedResourceQuotas: []kubermaticv1.ResourceQuota{
				*genResourceQuota(
					buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
					generator.GenDefaultProject().Name,
					*genResourceDetails("2", "5G", "10G"),
					true),
			},
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genSettings(genResourceDetails("2", "5G", "10G")), generator.GenDefaultProject()).
				Build(),
		},
		{
			name: "scenario 2: update default project quota",
			expectedResourceQuotas: []kubermaticv1.ResourceQuota{
				*genResourceQuota(
					buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
					generator.GenDefaultProject().Name,
					*genResourceDetails("2", "5G", "10G"),
					true),
			},
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genSettings(genResourceDetails("2", "5G", "10G")),
					generator.GenDefaultProject(),
					genResourceQuota(buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
						generator.GenDefaultProject().Name,
						*genResourceDetails("1", "3G", "7G"),
						true),
				).Build(),
		},
		{
			name: "scenario 3: dont update custom project quota",
			expectedResourceQuotas: []kubermaticv1.ResourceQuota{
				*genResourceQuota(
					buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
					generator.GenDefaultProject().Name,
					*genResourceDetails("1", "3G", "7G"),
					false),
			},
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genSettings(genResourceDetails("2", "5G", "10G")),
					generator.GenDefaultProject(),
					genResourceQuota(buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
						generator.GenDefaultProject().Name,
						*genResourceDetails("1", "3G", "7G"),
						false),
				).Build(),
		},
		{
			name:                   "scenario 4: delete default project quota",
			expectedResourceQuotas: []kubermaticv1.ResourceQuota{},
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genSettings(nil),
					generator.GenDefaultProject(),
					genResourceQuota(buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
						generator.GenDefaultProject().Name,
						*genResourceDetails("1", "3G", "7G"),
						true),
				).Build(),
		},
		{
			name: "scenario 5: dont delete custom project quota",
			expectedResourceQuotas: []kubermaticv1.ResourceQuota{
				*genResourceQuota(
					buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
					generator.GenDefaultProject().Name,
					*genResourceDetails("1", "3G", "7G"),
					false),
			},
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genSettings(nil),
					generator.GenDefaultProject(),
					genResourceQuota(buildNameFromSubject(kubermaticv1.Subject{Name: generator.GenDefaultProject().Name, Kind: kubermaticv1.ProjectSubjectKind}),
						generator.GenDefaultProject().Name,
						*genResourceDetails("1", "3G", "7G"),
						false),
				).Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: kubermaticv1.GlobalSettingsName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rqs := &kubermaticv1.ResourceQuotaList{}
			err := tc.masterClient.List(ctx, rqs)
			if err != nil {
				t.Fatalf("failed to get resource quotas: %v", err)
			}

			resultRqs := []kubermaticv1.ResourceQuota{}
			// remove resource version
			for _, rq := range rqs.Items {
				rq.SetResourceVersion("")
				rq.Kind = ""
				rq.APIVersion = ""
				resultRqs = append(resultRqs, rq)
			}

			if !diff.SemanticallyEqual(tc.expectedResourceQuotas, resultRqs) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedResourceQuotas, resultRqs))
			}
		})
	}
}

func genSettings(resourceDetails *kubermaticv1.ResourceDetails) *kubermaticv1.KubermaticSetting {
	s := &kubermaticv1.KubermaticSetting{}
	s.Name = kubermaticv1.GlobalSettingsName
	if resourceDetails != nil {
		s.Spec = kubermaticv1.SettingSpec{
			DefaultProjectResourceQuota: &kubermaticv1.DefaultProjectResourceQuota{
				Quota: *resourceDetails,
			},
		}
	}
	return s
}

func genResourceDetails(cpu, mem, storage string) *kubermaticv1.ResourceDetails {
	return kubermaticv1.NewResourceDetails(resource.MustParse(cpu), resource.MustParse(mem), resource.MustParse(storage))
}

func genResourceQuota(name, subjectName string, quota kubermaticv1.ResourceDetails, def bool) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: subjectName,
			Kind: kubermaticv1.ProjectSubjectKind,
		},
		Quota: quota,
	}
	if def {
		rq.Labels = map[string]string{
			DefaultProjectResourceQuotaKey: DefaultProjectResourceQuotaValue,
		}
	}

	return rq
}
