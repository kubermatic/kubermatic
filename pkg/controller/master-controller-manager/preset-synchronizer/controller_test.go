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

package presetsynchronizer

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
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const presetName = "preset-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name           string
		requestName    string
		expectedPreset *kubermaticv1.Preset
		masterClient   ctrlruntimeclient.Client
		seedClient     ctrlruntimeclient.Client
	}{
		{
			name:           "scenario 1: sync preset from master cluster to seed cluster",
			requestName:    presetName,
			expectedPreset: generatePreset(presetName, false),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generatePreset(presetName, false), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:           "scenario 2: cleanup preset on the seed cluster when master preset is being terminated",
			requestName:    presetName,
			expectedPreset: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generatePreset(presetName, true), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generatePreset(presetName, false), generator.GenTestSeed()).
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
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedPreset := &kubermaticv1.Preset{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedPreset)
			if tc.expectedPreset == nil {
				if err == nil {
					t.Fatal("failed clean up preset on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get template: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get template: %v", err)
				}

				seedPreset.ResourceVersion = ""
				seedPreset.APIVersion = ""
				seedPreset.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedPreset, seedPreset) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedPreset, seedPreset))
				}
			}
		})
	}
}

func generatePreset(name string, deleted bool) *kubermaticv1.Preset {
	pr := &kubermaticv1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PresetSpec{
			Fake: &kubermaticv1.Fake{
				Token: "fake",
			},
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		pr.DeletionTimestamp = &deleteTime
		pr.Finalizers = append(pr.Finalizers, cleanupFinalizer)
	}
	return pr
}
