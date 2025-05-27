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

package mastercontroller

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

const rqName = "resourceQuota"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name          string
		requestName   string
		expectedUsage kubermaticv1.ResourceDetails
		masterClient  ctrlruntimeclient.Client
		seedClients   map[string]ctrlruntimeclient.Client
	}{
		{
			name:          "scenario 1: calculate rq global usage",
			requestName:   rqName,
			expectedUsage: *genResourceDetails("7", "7G", "18G"),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genResourceQuota(rqName, kubermaticv1.ResourceDetails{}), generator.GenTestSeed()).
				Build(),
			seedClients: map[string]ctrlruntimeclient.Client{
				"first": fake.
					NewClientBuilder().
					WithObjects(genResourceQuota(rqName, *genResourceDetails("2", "5G", "10G"))).
					Build(),
				"second": fake.
					NewClientBuilder().
					WithObjects(genResourceQuota(rqName, *genResourceDetails("5", "2G", "8G"))).
					Build(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  tc.seedClients,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err := tc.masterClient.Get(ctx, request.NamespacedName, rq)

			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedUsage, rq.Status.GlobalUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUsage, rq.Status.GlobalUsage))
			}
		})
	}
}

func genResourceQuota(name string, localUsage kubermaticv1.ResourceDetails) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: "project1",
			Kind: "project",
		},
	}

	rq.Status.LocalUsage = localUsage
	return rq
}

func genResourceDetails(cpu, mem, storage string) *kubermaticv1.ResourceDetails {
	return kubermaticv1.NewResourceDetails(resource.MustParse(cpu), resource.MustParse(mem), resource.MustParse(storage))
}
