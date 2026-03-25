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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateUserProjectBinding(userProjectBindingName, false), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:                       "scenario 2: cleanup userProjectBinding on the seed cluster when master userProjectBinding is being terminated",
			requestName:                userProjectBindingName,
			expectedUserProjectBinding: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateUserProjectBinding(userProjectBindingName, true), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generateUserProjectBinding(userProjectBindingName, false), generator.GenTestSeed()).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &events.FakeRecorder{},
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
		userProjectBinding.Finalizers = append(userProjectBinding.Finalizers, cleanupFinalizer)
	}
	return userProjectBinding
}
