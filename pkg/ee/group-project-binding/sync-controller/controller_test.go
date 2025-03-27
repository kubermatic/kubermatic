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

package synccontroller

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const groupProjectBindingName = "group-project-binding-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                        string
		requestName                 string
		existingMasterResources     []ctrlruntimeclient.Object
		existingSeedResources       []ctrlruntimeclient.Object
		expectedGroupProjectBinding *kubermaticv1.GroupProjectBinding
	}{
		{
			name:        "scenario 1: sync groupProjectBinding from master cluster to seed cluster",
			requestName: groupProjectBindingName,
			existingMasterResources: []ctrlruntimeclient.Object{
				generateGroupProjectBinding(groupProjectBindingName, false),
			},
			expectedGroupProjectBinding: generateGroupProjectBinding(groupProjectBindingName, false),
		},
		{
			name:        "scenario 2: cleanup groupProjectBinding on the seed cluster when master groupProjectBinding is being terminated",
			requestName: groupProjectBindingName,
			existingMasterResources: []ctrlruntimeclient.Object{
				generateGroupProjectBinding(groupProjectBindingName, true),
			},
			existingSeedResources: []ctrlruntimeclient.Object{
				generateGroupProjectBinding(groupProjectBindingName, false),
			},
			expectedGroupProjectBinding: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			seed := generator.GenTestSeed()

			masterClient := fake.NewClientBuilder().
				WithObjects(seed).
				WithObjects(tc.existingMasterResources...).
				Build()

			seedClient := fake.NewClientBuilder().
				WithObjects(seed).
				WithObjects(tc.existingSeedResources...).
				Build()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: masterClient,
				seedsGetter: func() (map[string]*kubermaticv1.Seed, error) {
					return map[string]*kubermaticv1.Seed{
						seed.Name: seed,
					}, nil
				},
				seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return seedClient, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedGroupProjectBinding := &kubermaticv1.GroupProjectBinding{}
			err := seedClient.Get(ctx, request.NamespacedName, seedGroupProjectBinding)
			if tc.expectedGroupProjectBinding == nil {
				if err == nil {
					t.Fatal("failed clean up groupProjectBinding on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get groupProjectBinding: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get groupProjectBinding: %v", err)
				}

				seedGroupProjectBinding.ResourceVersion = ""
				seedGroupProjectBinding.APIVersion = ""
				seedGroupProjectBinding.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedGroupProjectBinding, seedGroupProjectBinding) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedGroupProjectBinding, seedGroupProjectBinding))
				}
			}
		})
	}
}

func generateGroupProjectBinding(name string, deleted bool) *kubermaticv1.GroupProjectBinding {
	groupProjectBinding := &kubermaticv1.GroupProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.GroupProjectBindingSpec{
			ProjectID: "test-project",
			Group:     "test",
			Role:      rbac.EditorGroupNamePrefix + "test-project",
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		groupProjectBinding.DeletionTimestamp = &deleteTime
		groupProjectBinding.Finalizers = append(groupProjectBinding.Finalizers, cleanupFinalizer)
	}
	return groupProjectBinding
}
