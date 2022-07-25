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

package labelownercontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		name                       string
		requestName                string
		expectedLabels             map[string]string
		expectedReconcileErrStatus metav1.StatusReason
		masterClient               ctrlruntimeclient.Client
	}{
		{
			name:        "scenario 1: reconcile labels and owner ref",
			requestName: rqName,
			expectedLabels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: test.GenDefaultProject().Name,
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
			},
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genResourceQuota(rqName, kubermaticv1.ResourceDetails{}), test.GenTestSeed(), test.GenDefaultProject()).
				Build(),
		},
		{
			name:                       "scenario 2: error when subject project is not present",
			requestName:                rqName,
			expectedReconcileErrStatus: metav1.StatusReasonNotFound,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(genResourceQuota(rqName, kubermaticv1.ResourceDetails{}), test.GenTestSeed()).
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
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			_, err := r.Reconcile(ctx, request)
			if tc.expectedReconcileErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s", tc.expectedReconcileErrStatus)
				}

				if tc.expectedReconcileErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedReconcileErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err = tc.masterClient.Get(ctx, request.NamespacedName, rq)
			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedLabels, rq.Labels) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedLabels, rq.Labels))
			}
			if len(rq.OwnerReferences) != 1 {
				t.Fatal("expected owner reference, got none")
			}
		})
	}
}

func genResourceQuota(name string, localUsage kubermaticv1.ResourceDetails) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: test.GenDefaultProject().Name,
			Kind: kubermaticv1.ProjectSubjectKind,
		},
	}

	return rq
}
