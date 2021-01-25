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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {

	tests := []struct {
		name                      string
		clusterName               string
		existingKubermaticObjects []runtime.Object
	}{
		{
			name:        "scenario 1: cleanup finalizer and kubeconfig secret",
			clusterName: "test",
			existingKubermaticObjects: []runtime.Object{
				genExternalCluster("test", metav1.Now()),
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: genExternalCluster("test", metav1.Now()).GetKubeconfigSecretName(), Namespace: resources.KubermaticNamespace},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario

			kubermaticFakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingKubermaticObjects...)

			// act
			ctx := context.Background()
			target := Reconciler{
				Client: kubermaticFakeClient,
				log:    kubermaticlog.Logger,
			}

			_, err := target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: test.clusterName}})

			// validate
			if err != nil {
				t.Fatal(err)
			}
			cluster := &kubermaticv1.ExternalCluster{}
			err = kubermaticFakeClient.Get(ctx, client.ObjectKey{Name: test.clusterName}, cluster)
			if err != nil {
				t.Fatal(err)
			}
			if kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.ExternalClusterKubeconfigCleanupFinalizer) {
				t.Fatal("the finalizer should be deleted")
			}

			secretKubeconfig := &corev1.Secret{}
			err = kubermaticFakeClient.Get(ctx, client.ObjectKey{Name: cluster.GetKubeconfigSecretName()}, secretKubeconfig)
			if err == nil {
				t.Fatal("expected error")
			}
			if !kerrors.IsNotFound(err) {
				t.Fatalf("expected not-found error, but got %v", err)
			}
		})
	}
}

func genExternalCluster(name string, deletionTimestamp metav1.Time) *kubermaticv1.ExternalCluster {

	cluster := &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, DeletionTimestamp: &deletionTimestamp},
		Spec: kubermaticv1.ExternalClusterSpec{
			HumanReadableName: name,
		},
	}

	kuberneteshelper.AddFinalizer(cluster, kubermaticapiv1.ExternalClusterKubeconfigCleanupFinalizer)

	cluster.Spec.KubeconfigReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Namespace: resources.KubermaticNamespace,
			Name:      cluster.GetKubeconfigSecretName(),
		},
	}

	return cluster
}
