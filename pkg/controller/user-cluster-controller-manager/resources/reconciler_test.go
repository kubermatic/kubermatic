/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package resources

import (
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureExternalAdminUserIsRemoved(t *testing.T) {
	externalAdminUserServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: resources.UserClusterLegacyExternalAdminUserServiceAccountName, Namespace: metav1.NamespaceSystem},
	}
	externalAdminUserClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: resources.UserClusterLegacyExternalAdminUserClusterRoleBindingName},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "cluster-admin"},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      resources.UserClusterLegacyExternalAdminUserServiceAccountName,
			Namespace: metav1.NamespaceSystem,
		}},
	}
	// Objects with the same name in another namespace, or unrelated objects
	// in kube-system, must survive the cleanup.
	unrelatedObjects := []ctrlruntimeclient.Object{
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: resources.UserClusterLegacyExternalAdminUserServiceAccountName, Namespace: metav1.NamespaceDefault},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: metav1.NamespaceSystem},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "system:coredns"},
		},
	}

	testCases := []struct {
		name    string
		objects []ctrlruntimeclient.Object
	}{
		{
			name:    "ServiceAccount and ClusterRoleBinding exist",
			objects: []ctrlruntimeclient.Object{externalAdminUserServiceAccount, externalAdminUserClusterRoleBinding},
		},
		{
			name:    "only the ClusterRoleBinding exists",
			objects: []ctrlruntimeclient.Object{externalAdminUserClusterRoleBinding},
		},
		{
			name:    "only the ServiceAccount exists",
			objects: []ctrlruntimeclient.Object{externalAdminUserServiceAccount},
		},
		{
			name: "nothing to remove",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := corev1.AddToScheme(scheme); err != nil {
				t.Fatalf("add core API to scheme: %v", err)
			}
			if err := rbacv1.AddToScheme(scheme); err != nil {
				t.Fatalf("add rbac API to scheme: %v", err)
			}

			var objects []ctrlruntimeclient.Object
			for _, obj := range append(tc.objects, unrelatedObjects...) {
				objects = append(objects, obj.DeepCopyObject().(ctrlruntimeclient.Object))
			}
			client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &reconciler{Client: client}

			ctx := context.Background()
			if err := r.ensureExternalAdminUserIsRemoved(ctx); err != nil {
				t.Fatalf("ensureExternalAdminUserIsRemoved() error = %v", err)
			}

			assertNotFound(t, client, types.NamespacedName{Name: resources.UserClusterLegacyExternalAdminUserClusterRoleBindingName}, &rbacv1.ClusterRoleBinding{})
			assertNotFound(t, client, types.NamespacedName{Name: resources.UserClusterLegacyExternalAdminUserServiceAccountName, Namespace: metav1.NamespaceSystem}, &corev1.ServiceAccount{})

			for _, obj := range unrelatedObjects {
				key := ctrlruntimeclient.ObjectKeyFromObject(obj)
				if err := client.Get(ctx, key, obj.DeepCopyObject().(ctrlruntimeclient.Object)); err != nil {
					t.Errorf("expected unrelated %T %s to be kept, got error: %v", obj, key, err)
				}
			}

			// A second run must be a no-op.
			if err := r.ensureExternalAdminUserIsRemoved(ctx); err != nil {
				t.Fatalf("second ensureExternalAdminUserIsRemoved() error = %v", err)
			}
		})
	}
}

func assertNotFound(t *testing.T, client ctrlruntimeclient.Client, key types.NamespacedName, obj ctrlruntimeclient.Object) {
	t.Helper()

	if err := client.Get(context.Background(), key, obj); !apierrors.IsNotFound(err) {
		t.Errorf("expected %T %s to be removed, got error: %v", obj, key, err)
	}
}
