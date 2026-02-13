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

package usersynchronizer

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
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

const userName = "user-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name         string
		requestName  string
		expectedUser *kubermaticv1.User
		masterClient ctrlruntimeclient.Client
		seedClient   ctrlruntimeclient.Client
	}{
		{
			name:         "scenario 1: sync user from master cluster to seed cluster",
			requestName:  userName,
			expectedUser: generateUser(userName, false),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateUser(userName, false)).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:         "scenario 2: cleanup user on the seed cluster when master user is being terminated",
			requestName:  userName,
			expectedUser: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateUser(userName, true)).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generateUser(userName, false)).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:             kubermaticlog.Logger,
				recorder:        &events.FakeRecorder{},
				masterClient:    tc.masterClient,
				seedClients:     map[string]ctrlruntimeclient.Client{"test": tc.seedClient},
				masterAPIReader: tc.masterClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedUser := &kubermaticv1.User{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedUser)
			if tc.expectedUser == nil {
				if err == nil {
					t.Fatal("failed clean up user on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get user: %v", err)
				}
			} else {
				seedUser = sanitize(seedUser)
				tc.expectedUser = sanitize(tc.expectedUser)

				if err != nil {
					t.Fatalf("failed to get user: %v", err)
				}

				seedUser.ResourceVersion = ""
				seedUser.APIVersion = ""
				seedUser.Kind = ""
				seedUser.UID = tc.expectedUser.UID

				if !diff.SemanticallyEqual(tc.expectedUser, seedUser) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUser, seedUser))
				}
			}
		})
	}
}

func generateUser(name string, deleted bool) *kubermaticv1.User {
	user := generator.GenDefaultUser()
	user.Name = name
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		user.DeletionTimestamp = &deleteTime
		user.Finalizers = append(user.Finalizers, cleanupFinalizer)
	}
	return user
}

// Skip problematic fields from the comparison such as `LastSeen` Time field
// as even DeepCopy does not result in equal internal location field value.
func sanitize(user *kubermaticv1.User) *kubermaticv1.User {
	user.Status.LastSeen = metav1.Time{}
	return user
}
