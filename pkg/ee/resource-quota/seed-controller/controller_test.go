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

package seedcontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const rqName = "resourceQuota"
const projectID = "project1"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name          string
		requestName   string
		resourceQuota *kubermaticv1.ResourceQuota
		seedClient    ctrlruntimeclient.Client
		expectedUsage kubermaticv1.ResourceDetails
	}{
		{
			name:          "scenario 1: calculate rq local usage",
			requestName:   rqName,
			resourceQuota: genResourceQuota(rqName),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genResourceQuota(rqName),
					genCluster("c1", projectID, "2", "5G", "10G"),
					genCluster("c2", projectID, "5", "2G", "8G"),
					genCluster("notSameProjectCluster", "impostor", "3", "3G", "3G")).
				Build(),
			expectedUsage: *genResourceDetails("7", "7G", "18G"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:        kubermaticlog.Logger,
				recorder:   &events.FakeRecorder{},
				seedClient: tc.seedClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, rq)

			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedUsage, rq.Status.LocalUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUsage, rq.Status.LocalUsage))
			}
		})
	}
}

func genResourceQuota(name string) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: projectID,
			Kind: kubermaticv1.ProjectSubjectKind,
		},
	}

	return rq
}

func genResourceDetails(cpu, mem, storage string) *kubermaticv1.ResourceDetails {
	return kubermaticv1.NewResourceDetails(resource.MustParse(cpu), resource.MustParse(mem), resource.MustParse(storage))
}

func genCluster(name, projectID, cpu, mem, storage string) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{}
	cluster.Name = name
	cluster.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
	cluster.Status.ResourceUsage = genResourceDetails(cpu, mem, storage)

	return cluster
}
