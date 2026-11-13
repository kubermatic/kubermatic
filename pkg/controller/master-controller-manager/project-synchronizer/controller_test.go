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

package projectsynchronizer

import (
	"context"
	"fmt"
	"testing"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const projectName = "project-test"

var projectLabels = map[string]string{
	"test":        "project",
	"description": "test",
}

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name            string
		requestName     string
		expectedProject *kubermaticv1.Project
		masterClient    ctrlruntimeclient.Client
		seedClient      ctrlruntimeclient.Client
	}{
		{
			name:            "scenario 1: sync project from master cluster to seed cluster",
			requestName:     projectName,
			expectedProject: generateProject(projectName, false, nil),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateProject(projectName, false, nil), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:            "scenario 2: cleanup project on the seed cluster when master project is being terminated",
			requestName:     projectName,
			expectedProject: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateProject(projectName, true, nil), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generateProject(projectName, false, nil), generator.GenTestSeed()).
				Build(),
		},
		{
			name:            "scenario 3: sync project with labels from master cluster to seed cluster",
			requestName:     projectName,
			expectedProject: generateProject(projectName, false, projectLabels),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateProject(projectName, false, projectLabels), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
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

			seedProject := &kubermaticv1.Project{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedProject)
			if tc.expectedProject == nil {
				if err == nil {
					t.Fatal("failed clean up project on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get project: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get project: %v", err)
				}

				seedProject.ResourceVersion = ""
				seedProject.APIVersion = ""
				seedProject.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedProject, seedProject) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedProject, seedProject))
				}
			}
		})
	}
}

func TestDeletionCleansUpClusterBackupStorageLocations(t *testing.T) {
	ctx := context.Background()

	projectCBSL := generateCBSL("project-location", projectName)
	otherCBSL := generateCBSL("other-location", "other-project")

	masterClient := fake.
		NewClientBuilder().
		WithObjects(generateProject(projectName, true, nil), projectCBSL, otherCBSL).
		Build()

	r := &reconciler{
		log:          kubermaticlog.Logger,
		recorder:     &events.FakeRecorder{},
		masterClient: masterClient,
		seedClients:  map[string]ctrlruntimeclient.Client{"test": fake.NewClientBuilder().Build()},
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: projectName}}
	if _, err := r.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}

	err := masterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(projectCBSL), &kubermaticv1.ClusterBackupStorageLocation{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected ClusterBackupStorageLocation of the deleted project to be deleted, but got: %v", err)
	}

	if err := masterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(otherCBSL), &kubermaticv1.ClusterBackupStorageLocation{}); err != nil {
		t.Fatalf("expected ClusterBackupStorageLocation of another project to remain, but got: %v", err)
	}
}

func generateCBSL(name, projectID string) *kubermaticv1.ClusterBackupStorageLocation {
	return &kubermaticv1.ClusterBackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kubermatic",
			Labels:    map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: velerov1.BackupStorageLocationSpec{
			Credential: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "credential-" + name},
			},
		},
	}
}

func generateProject(name string, deleted bool, labels map[string]string) *kubermaticv1.Project {
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: fmt.Sprintf("project-%s", name),
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		project.DeletionTimestamp = &deleteTime
		project.Finalizers = append(project.Finalizers, cleanupFinalizer)
	}
	return project
}
