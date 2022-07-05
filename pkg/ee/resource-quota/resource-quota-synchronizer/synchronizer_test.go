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

package resourcequotasynchronizer

import (
	"context"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const rqName = "resourceQuota"

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = kubermaticv1.AddToScheme(scheme)

	testCases := []struct {
		name                 string
		requestName          string
		expectedRQ           *kubermaticv1.ResourceQuota
		expectedGetErrStatus metav1.StatusReason
		masterClient         ctrlruntimeclient.Client
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name:        "scenario 1: sync rq to seed cluster",
			requestName: rqName,
			expectedRQ:  genResourceQuota(rqName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genResourceQuota(rqName, false), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				Build(),
		},
		{
			name:                 "scenario 2: cleanup rq on seed cluster when master rq is being terminated",
			requestName:          rqName,
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genResourceQuota(rqName, true), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genResourceQuota(rqName, false)).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName, Namespace: kubermaticresources.KubermaticNamespace}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, rq)
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got rq: %v", tc.expectedGetErrStatus, rq)
				}

				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			rq.ResourceVersion = ""
			rq.APIVersion = ""
			rq.Kind = ""

			// the local usage must NOT be identical, as it's not supposed to be synced
			if diff.SemanticallyEqual(tc.expectedRQ.Status.LocalUsage, rq.Status.LocalUsage) {
				t.Fatal("LocalUsage should not have been synchronized.")
			}

			// to make equivalence checks easier, let's just fake the LocalUsage
			rq.Status.LocalUsage = tc.expectedRQ.Status.LocalUsage

			if !diff.SemanticallyEqual(tc.expectedRQ, rq) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedRQ, rq))
			}
		})
	}
}

func genResourceQuota(name string, deleted bool) *kubermaticv1.ResourceQuota {
	cpu := resource.MustParse("5")
	mem := resource.MustParse("5G")
	storage := resource.MustParse("10G")

	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Namespace = kubermaticresources.KubermaticNamespace
	rq.Labels = map[string]string{
		kubermaticv1.ResourceQuotaSubjectNameLabelKey: "project1",
		kubermaticv1.ResourceQuotaSubjectKindLabelKey: "project",
	}
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: "project1",
			Kind: "project",
		},
		Quota: kubermaticv1.ResourceDetails{
			CPU:     &cpu,
			Memory:  &mem,
			Storage: &storage,
		},
	}
	rq.Status = kubermaticv1.ResourceQuotaStatus{
		GlobalUsage: kubermaticv1.ResourceDetails{
			CPU:     &cpu,
			Memory:  &mem,
			Storage: &storage,
		},
		LocalUsage: kubermaticv1.ResourceDetails{
			CPU:     &cpu,
			Memory:  &mem,
			Storage: &storage,
		},
	}

	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		rq.DeletionTimestamp = &deleteTime
		rq.Finalizers = append(rq.Finalizers, apiv1.ResourceQuotaSeedCleanupFinalizer)
	}

	return rq
}
