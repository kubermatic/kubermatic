/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package groupprojectbindingsynchronizer

import (
	"context"
	"reflect"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
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

const groupProjectBindingName = "user-project-binding-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                        string
		requestName                 string
		expectedGroupProjectBinding *kubermaticv1.GroupProjectBinding
		masterClient                ctrlruntimeclient.Client
		seedClient                  ctrlruntimeclient.Client
	}{
		{
			name:                        "scenario 1: sync groupProjectBinding from master cluster to seed cluster",
			requestName:                 groupProjectBindingName,
			expectedGroupProjectBinding: generateGroupProjectBinding(groupProjectBindingName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateGroupProjectBinding(groupProjectBindingName, false), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build(),
		},
		{
			name:                        "scenario 2: cleanup groupProjectBinding on the seed cluster when master groupProjectBinding is being terminated",
			requestName:                 groupProjectBindingName,
			expectedGroupProjectBinding: nil,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateGroupProjectBinding(groupProjectBindingName, true), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateGroupProjectBinding(groupProjectBindingName, false), test.GenTestSeed()).
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

			seedGroupProjectBinding := &kubermaticv1.GroupProjectBinding{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedGroupProjectBinding)
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
				if !reflect.DeepEqual(seedGroupProjectBinding.Spec, tc.expectedGroupProjectBinding.Spec) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedGroupProjectBinding, tc.expectedGroupProjectBinding))
				}
				if !reflect.DeepEqual(seedGroupProjectBinding.Name, tc.expectedGroupProjectBinding.Name) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedGroupProjectBinding, tc.expectedGroupProjectBinding))
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
		groupProjectBinding.Finalizers = append(groupProjectBinding.Finalizers, apiv1.SeedGroupProjectBindingCleanupFinalizer)
	}
	return groupProjectBinding
}
