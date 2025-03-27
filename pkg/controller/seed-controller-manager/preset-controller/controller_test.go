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

package presetcontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var now = metav1.Now()

func TestReconcile(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedClusters     []string
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: reconcile only deleted preset",
			namespacedName: types.NamespacedName{
				Name: generator.TestFakeCredential,
			},
			expectedClusters: nil,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(
					getPreset(nil),
					genCluster("ct2-0", "bob@acme.com", generator.TestFakeCredential),
					genCluster("ct2-1", "bob@acme.com", "test"),
					genCluster("ct2-2", "bob@acme.com", generator.TestFakeCredential),
				).
				Build(),
		},
		{
			name: "scenario 2: reconcile deleted preset",
			namespacedName: types.NamespacedName{
				Name: generator.TestFakeCredential,
			},
			expectedClusters: []string{"ct2-0", "ct2-2"},
			seedClient: fake.
				NewClientBuilder().
				WithObjects(
					getPreset(&now),
					genCluster("ct2-0", "bob@acme.com", generator.TestFakeCredential),
					genCluster("ct2-1", "bob@acme.com", "test"),
					genCluster("ct2-2", "bob@acme.com", generator.TestFakeCredential),
				).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				seedClient:              tc.seedClient,
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %v", tc.expectedGetErrStatus)
				}
				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			for _, clusterName := range tc.expectedClusters {
				cluster := &kubermaticv1.Cluster{}
				if err := tc.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterName}, cluster); err != nil {
					t.Fatalf("can't get expected cluster %s", clusterName)
				}
				if cluster.Annotations == nil {
					t.Fatal("expected annotations for the cluster")
				}
				if cluster.Annotations[kubermaticv1.PresetInvalidatedAnnotation] != string(kubermaticv1.PresetDeleted) {
					t.Fatalf("expected annotation %s with value %s", kubermaticv1.PresetInvalidatedAnnotation, kubermaticv1.PresetDeleted)
				}
			}
		})
	}
}

func genCluster(name, userEmail, preset string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{kubermaticv1.IsCredentialPresetLabelKey: "true"},
			ResourceVersion: "1",
			Finalizers:      []string{kubermaticv1.CredentialsSecretsCleanupFinalizer},
			Annotations:     map[string]string{kubermaticv1.ClusterTemplateUserAnnotationKey: userEmail, kubermaticv1.PresetNameAnnotation: preset},
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "fake-dc",
				Fake:           &kubermaticv1.FakeCloudSpec{},
			},
		},
		Status: kubermaticv1.ClusterStatus{
			UserEmail: userEmail,
		},
	}
}

func getPreset(deletionTimestamp *metav1.Time) *kubermaticv1.Preset {
	preset := generator.GenDefaultPreset()
	preset.DeletionTimestamp = deletionTimestamp
	if deletionTimestamp != nil {
		preset.Finalizers = []string{"dummy"}
	}

	return preset
}
