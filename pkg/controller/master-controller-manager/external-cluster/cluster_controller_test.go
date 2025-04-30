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

package externalcluster

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	tests := []struct {
		name                      string
		clusterName               string
		isDelete                  bool
		existingKubermaticObjects []ctrlruntimeclient.Object
	}{
		{
			name:        "scenario 1: cleanup finalizer and kubeconfig secret",
			clusterName: "test",
			isDelete:    true,
			existingKubermaticObjects: []ctrlruntimeclient.Object{
				genExternalCluster("test", true),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      genExternalCluster("test", true).GetKubeconfigSecretName(),
						Namespace: resources.KubermaticNamespace,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			kubermaticFakeClient := fake.
				NewClientBuilder().
				WithObjects(test.existingKubermaticObjects...).
				Build()

			// act
			ctx := context.Background()
			target := Reconciler{
				Client: kubermaticFakeClient,
				log:    kubermaticlog.Logger,
			}

			// finalizers are removed step by step and this takes multiple reconciliations
			if _, err := target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: test.clusterName}}); err != nil {
				t.Fatal(err)
			}

			if _, err := target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: test.clusterName}}); err != nil {
				t.Fatal(err)
			}

			// ensure the ExternalCluster is gone (the controller removed the finalizer, and since a
			// DeletionTimestamp was set, it should now be gone)
			cluster := &kubermaticv1.ExternalCluster{}
			err := kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: test.clusterName}, cluster)
			if err == nil {
				t.Fatal("expected ExternalCluster to be gone, but found it anyway")
			}
			if !apierrors.IsNotFound(err) {
				t.Fatalf("expected not-found error, but got %v", err)
			}

			// the secret should also be gone
			secretKubeconfig := &corev1.Secret{}
			err = kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: cluster.GetKubeconfigSecretName()}, secretKubeconfig)
			if err == nil {
				t.Fatal("expected secret to be gone, but found it anyway")
			}
			if !apierrors.IsNotFound(err) {
				t.Fatalf("expected not-found error, but got %v", err)
			}
		})
	}
}

func genExternalCluster(name string, isDelete bool) *kubermaticv1.ExternalCluster {
	cluster := &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ExternalClusterSpec{
			HumanReadableName: name,
			CloudSpec:         kubermaticv1.ExternalClusterCloudSpec{},
		},
	}
	kuberneteshelper.AddFinalizer(cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer)
	kuberneteshelper.AddFinalizer(cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer)

	if isDelete {
		deletionTimestamp := metav1.Now()
		cluster.DeletionTimestamp = &deletionTimestamp
	}
	cluster.Spec.KubeconfigReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Namespace: resources.KubermaticNamespace,
			Name:      cluster.GetKubeconfigSecretName(),
		},
	}

	return cluster
}
