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

package applicationsecretsynchronizer

import (
	"context"
	"reflect"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const secretName = "secret-1"
const seedNamespace = "kubermatic"
const masterNamespace = "master" // set this to something other than seedNamespace, to ensure we test the namespace override

func TestReconcile(t *testing.T) {
	masterSecret := generateSecret(secretName, masterNamespace)
	seedSecret := generateSecret(secretName, seedNamespace)

	testCases := []struct {
		name         string
		masterClient ctrlruntimeclient.Client
		seedClient   ctrlruntimeclient.Client
		expSecret    *corev1.Secret
	}{
		{
			name:         "scenario 1: secret in master, but not in seed",
			masterClient: fake.NewClientBuilder().WithObjects(masterSecret).Build(),
			seedClient:   fake.NewClientBuilder().Build(),
			expSecret:    seedSecret,
		},
		{
			name:         "scenario 2: secret not in master, but still in seed",
			masterClient: fake.NewClientBuilder().Build(),
			seedClient:   fake.NewClientBuilder().WithObjects(seedSecret).Build(),
			expSecret:    nil,
		},
		{
			name:         "scenario 3: secret not in master and it was never in seed",
			masterClient: fake.NewClientBuilder().Build(),
			seedClient:   fake.NewClientBuilder().Build(),
			expSecret:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &events.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
				namespace:    seedNamespace,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: masterSecret.Name, Namespace: masterSecret.Namespace}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			resSecret := &corev1.Secret{}
			err := tc.seedClient.Get(ctx, types.NamespacedName{Name: seedSecret.Name, Namespace: seedSecret.Namespace}, resSecret)

			if err != nil {
				if apierrors.IsNotFound(err) && tc.expSecret == nil {
					return
				}
				t.Fatalf("could not fetch result secret: %q", err)
			}

			if resSecret.Name != tc.expSecret.Name {
				t.Errorf("expected secret name to be %q, got %q", tc.expSecret.Name, resSecret.Name)
			}
			if resSecret.Namespace != tc.expSecret.Namespace {
				t.Errorf("expected secret namespace to be %q, got %q", tc.expSecret.Namespace, resSecret.Namespace)
			}
			if !reflect.DeepEqual(resSecret.Data, tc.expSecret.Data) {
				t.Errorf("expected secret data to be %q, got %q", tc.expSecret.Data, resSecret.Data)
			}
		})
	}
}

func generateSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"testkey": "testval",
		},
	}
}
