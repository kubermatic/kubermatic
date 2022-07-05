/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package userprojectbindingsynchronizer

import (
	"context"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

const userProjectBindingName = "user-project-binding-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                       string
		requestName                string
		expectedUserProjectBinding *kubermaticv1.UserProjectBinding
		masterClient               ctrlruntimeclient.Client
		seedClient                 ctrlruntimeclient.Client
	}{
		{
			name:                       "scenario 1: sync userProjectBinding from master cluster to seed cluster",
			requestName:                userProjectBindingName,
			expectedUserProjectBinding: generateUserProjectBinding(userProjectBindingName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateUserProjectBinding(userProjectBindingName, false), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build(),
		},
		{
			name:                       "scenario 2: cleanup userProjectBinding on the seed cluster when master userProjectBinding is being terminated",
			requestName:                userProjectBindingName,
			expectedUserProjectBinding: nil,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateUserProjectBinding(userProjectBindingName, true), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateUserProjectBinding(userProjectBindingName, false), test.GenTestSeed()).
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
				seedClients:  map[string]ctrlruntimeclient.Client{"test": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedUserProjectBinding := &kubermaticv1.UserProjectBinding{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedUserProjectBinding)
			if tc.expectedUserProjectBinding == nil {
				if err == nil {
					t.Fatal("failed clean up userProjectBinding on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get userProjectBinding: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get userProjectBinding: %v", err)
				}

				seedUserProjectBinding.ResourceVersion = ""
				seedUserProjectBinding.APIVersion = ""
				seedUserProjectBinding.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedUserProjectBinding, seedUserProjectBinding) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUserProjectBinding, seedUserProjectBinding))
				}
			}
		})
	}
}

func generateUserProjectBinding(name string, deleted bool) *kubermaticv1.UserProjectBinding {
	userProjectBinding := &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: "test-project",
			Group:     rbac.EditorGroupNamePrefix + "test-project",
			UserEmail: "test@test.com",
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		userProjectBinding.DeletionTimestamp = &deleteTime
		userProjectBinding.Finalizers = append(userProjectBinding.Finalizers, apiv1.SeedUserProjectBindingCleanupFinalizer)
	}
	return userProjectBinding
}
