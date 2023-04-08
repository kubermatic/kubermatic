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

package mlacontroller

import (
	"context"
	"testing"

	"go.uber.org/zap"

	sdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestGrafanaUserReconciler(objects []ctrlruntimeclient.Object) (*grafanaUserReconciler, *grafana.FakeGrafana) {
	dynamicClient := ctrlruntimefakeclient.
		NewClientBuilder().
		WithObjects(objects...).
		Build()

	gClient := grafana.NewFakeClient()

	// for these tests it's important to deal with a pre-existing magic default
	// org in which all new users are placed
	if err := gClient.CreateDefaultOrg(sdk.Org{Name: "grafana-default-org"}); err != nil {
		panic(err)
	}

	// expect that the org reconciler has created the KKP org in Grafana
	if _, err := gClient.CreateOrg(context.Background(), sdk.Org{Name: GrafanaOrganization}); err != nil {
		panic(err)
	}

	return &grafanaUserReconciler{
		seedClient: dynamicClient,
		log:        zap.NewNop().Sugar(),
		recorder:   record.NewFakeRecorder(10),
		clientProvider: func(ctx context.Context) (grafana.Client, error) {
			return gClient, nil
		},
	}, gClient
}

func TestGrafanaUserReconcile(t *testing.T) {
	ctx := context.Background()

	userName := "testuser"
	userEmail := "user@example.com"

	testCases := []struct {
		name      string
		objects   []ctrlruntimeclient.Object
		assertion func(t *testing.T, user *kubermaticv1.User, gClient *grafana.FakeGrafana, reconciler *grafanaUserReconciler, reconcileErr error)
	}{
		{
			name: "User added",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: userName,
					},
					Spec: kubermaticv1.UserSpec{
						Email:   userEmail,
						IsAdmin: false,
					},
				},
			},
			assertion: func(t *testing.T, user *kubermaticv1.User, gClient *grafana.FakeGrafana, reconciler *grafanaUserReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(user, mlaFinalizer) {
					t.Error("Expected user to have MLA finalizer, but does not.")
				}

				gUser, err := gClient.LookupUser(ctx, userEmail)
				if err != nil {
					t.Fatalf("User does not exist in Grafana: %v", err)
				}

				if gUser.Email != userEmail {
					t.Fatalf("User should have e-mail address %q, but has %q", userEmail, gUser.Email)
				}

				// user should not exist in default org anymore
				orgUsers, err := gClient.GetOrgUsers(ctx, gClient.Database.DefaultOrg)
				if err != nil {
					t.Fatalf("Failed to get users in default org: %v", err)
				}

				if len(orgUsers) != 0 {
					t.Fatalf("Default org should not have new user anymore, but has %+v", orgUsers)
				}

				// ... but should be in the KKP org
				org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
				if err != nil {
					t.Fatalf("Failed to get KKP org: %v", err)
				}

				orgUsers, err = gClient.GetOrgUsers(ctx, org.ID)
				if err != nil {
					t.Fatalf("Failed to get users in KKP org: %v", err)
				}

				if len(orgUsers) != 1 {
					t.Fatalf("KKP org should have 1 user, but has %+v", orgUsers)
				}

				expectedRole := getRoleForUser(user)
				if orgUsers[0].Role != string(expectedRole) {
					t.Fatalf("Expected user to have %q role in org, but has %q.", expectedRole, orgUsers[0].Role)
				}
			},
		},
		{
			name: "User IsAdmin updated to True",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: userName,
					},
					Spec: kubermaticv1.UserSpec{
						Email:   userEmail,
						IsAdmin: true,
					},
				},
			},
			assertion: func(t *testing.T, user *kubermaticv1.User, gClient *grafana.FakeGrafana, reconciler *grafanaUserReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(user, mlaFinalizer) {
					t.Error("Expected user to have MLA finalizer, but does not.")
				}

				user.Spec.IsAdmin = true
				if err := reconciler.seedClient.Update(ctx, user); err != nil {
					t.Fatalf("Failed to toggle isAdmin flag: %v", err)
				}

				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: userName}}
				_, reconcileErr = reconciler.Reconcile(ctx, request)
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile a second time: %v", reconcileErr)
				}

				// ... but should be in the KKP org
				org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
				if err != nil {
					t.Fatalf("Failed to get KKP org: %v", err)
				}

				orgUsers, err := gClient.GetOrgUsers(ctx, org.ID)
				if err != nil {
					t.Fatalf("Failed to get users in KKP org: %v", err)
				}

				if len(orgUsers) != 1 {
					t.Fatalf("KKP org should have 1 user, but has %+v", orgUsers)
				}

				expectedRole := getRoleForUser(user)
				if orgUsers[0].Role != string(expectedRole) {
					t.Fatalf("Expected user to have %q role in org, but has %q.", expectedRole, orgUsers[0].Role)
				}
			},
		},
		{
			name: "User delete",
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name:       userName,
						Finalizers: []string{mlaFinalizer, "just-a-test-do-not-delete-thanks"},
					},
					Spec: kubermaticv1.UserSpec{
						Email:   userEmail,
						IsAdmin: true,
					},
				},
			},
			assertion: func(t *testing.T, user *kubermaticv1.User, gClient *grafana.FakeGrafana, reconciler *grafanaUserReconciler, reconcileErr error) {
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile: %v", reconcileErr)
				}

				if !kubernetes.HasFinalizer(user, mlaFinalizer) {
					t.Error("Expected user to have MLA finalizer, but does not.")
				}

				if err := reconciler.seedClient.Delete(ctx, user); err != nil {
					t.Fatalf("Failed to delete user: %v", err)
				}

				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: userName}}
				_, reconcileErr = reconciler.Reconcile(ctx, request)
				if reconcileErr != nil {
					t.Fatalf("Failed to reconcile a second time: %v", reconcileErr)
				}

				if err := reconciler.seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(user), user); err != nil {
					t.Fatalf("failed to get user: %v", err)
				}

				if kubernetes.HasFinalizer(user, mlaFinalizer) {
					t.Error("Expected user not to have MLA finalizer, but does.")
				}

				if _, err := gClient.LookupUser(ctx, userEmail); err == nil {
					t.Fatal("User still exists in Grafana.")
				}
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reconciler, gClient := newTestGrafanaUserReconciler(tc.objects)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: userName}}
			_, reconcileErr := reconciler.Reconcile(ctx, request)

			user := &kubermaticv1.User{}
			if err := reconciler.seedClient.Get(ctx, request.NamespacedName, user); err != nil {
				t.Fatalf("failed to get user: %v", err)
			}

			tc.assertion(t, user, gClient, reconciler, reconcileErr)
		})
	}
}
