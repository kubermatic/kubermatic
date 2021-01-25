/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodecsrapprover

import (
	"context"
	"sync"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/certificates/v1beta1/fake"
	fakeclienttest "k8s.io/client-go/testing"

	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconciler_Reconcile(t *testing.T) {
	reaction := &fakeApproveReaction{}
	simpleReactor := &fakeclienttest.SimpleReactor{
		Verb:     "*",
		Resource: "*",
		Reaction: reaction.approveReaction,
	}
	testCases := []struct {
		name       string
		reconciler reconciler
	}{
		{
			name: "test approving a created certificate",
			reconciler: reconciler{
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client: ctrlruntimefakeclient.NewFakeClient(&certificatesv1beta1.CertificateSigningRequest{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "123456",
						Name:            "csr",
						Namespace:       metav1.NamespaceSystem,
					},
					Spec: certificatesv1beta1.CertificateSigningRequestSpec{
						Request: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQkdUQ0J2d0lCQURCZE1SVXdFd1lEVlFRS0V3eHplWE4wWlcwNmJtOWtaWE14UkRCQ0JnTlZCQU1UTzNONQpjM1JsYlRwdWIyUmxPbWRyWlMxcmRXSmxjbTFoZEdsakxXUmxkaTF3Y21WbGJYUnBZbXhsTFdKcFp5MWpZbVEyCk9EWmpaQzAxTkRsek1Ga3dFd1lIS29aSXpqMENBUVlJS29aSXpqMERBUWNEUWdBRWtDb2xGU24rZU10NDgxaWcKSmZUa3JjVmg1RGRhNnczZWkyYnBuMXgrRE9lc0VmdWR1Q3hLOWgyazN2L0RFb0J3eUpRNzF0RW5JbSs5UG04Lwp6dkcwa2FBQU1Bb0dDQ3FHU000OUJBTUNBMGtBTUVZQ0lRRGpSbjJJSjNNU2NKRXNSc3VHSTgyOFBvTW1TaWNuCkRqcjhKNWZ3QkxIeUxnSWhBTW9SY3FBREFYOTJHK0VhTGg5N1Q3NUo4em5mSUlVTE5ReW9OeTZVTUN5SgotLS0tLUVORCBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0K"),
						Usages: []certificatesv1beta1.KeyUsage{
							certificatesv1beta1.UsageDigitalSignature,
							certificatesv1beta1.UsageKeyEncipherment,
							certificatesv1beta1.UsageServerAuth,
						},
						Username: "test-user",
						Groups: []string{
							"system:nodes",
						},
					},
				}),
				certClient: &fake.FakeCertificateSigningRequests{
					Fake: &fake.FakeCertificatesV1beta1{
						Fake: &fakeclienttest.Fake{
							RWMutex:       sync.RWMutex{},
							ReactionChain: []fakeclienttest.Reactor{simpleReactor},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if err := tc.reconciler.reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "csr", Namespace: metav1.NamespaceSystem}}); err != nil {
				t.Fatalf("failed executing test: %v", err)
			}

			for _, cond := range reaction.expectedCSR.Status.Conditions {
				if cond.Type != certificatesv1beta1.CertificateApproved {
					t.Fatalf("failed updating csr condition")
				}
			}
		})
	}
}

type fakeApproveReaction struct {
	expectedCSR *certificatesv1beta1.CertificateSigningRequest
}

func (f *fakeApproveReaction) approveReaction(action fakeclienttest.Action) (bool, runtime.Object, error) {
	f.expectedCSR = action.(fakeclienttest.UpdateActionImpl).Object.(*certificatesv1beta1.CertificateSigningRequest)
	return true, f.expectedCSR, nil
}
