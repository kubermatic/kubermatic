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

package resources

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test"
)

func TestGetCSIMigrationFeatureGates(t *testing.T) {
	testCases := []struct {
		name             string
		cluster          *kubermaticv1.Cluster
		wantFeatureGates sets.String
	}{
		{
			name: "No CSI migration",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cluster-a",
					Annotations: map[string]string{},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.String{},
		},
		{
			name: "CSI migration",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.NewString("CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true"),
		},
		{
			name: "CSI migration completed",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
					Annotations: map[string]string{
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			wantFeatureGates: sets.NewString("CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true", "CSIMigrationOpenStackComplete=true"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			td := NewTemplateDataBuilder().
				WithCluster(tc.cluster).
				Build()
			if a, e := sets.NewString(td.GetCSIMigrationFeatureGates()...), tc.wantFeatureGates; !a.Equal(e) {
				t.Errorf("Want feature gates %v, but got %v", e, a)
			}
		})
	}
}

func TestGetControlPlaneComponentVersion(t *testing.T) {
	testCases := []struct {
		name           string
		cluster        *kubermaticv1.Cluster
		compDeployment *appsv1.Deployment
		kasDeployment  *appsv1.Deployment
		containerName  string
		wantVersion    *semver.Semver
		wantError      bool
	}{
		{
			name: "KAS current version match wanted version",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.18.1"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			compDeployment: test.NewDeploymentBuilder(test.NamespacedName{}).
				ContainerBuilder().
				WithName("scheduler").
				WithImage("k8s.gcr.io/scheduler:v1.17.2").
				AddContainer().Build(),
			kasDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: "apiserver", Namespace: "test"}).
				WithRolloutComplete().
				ContainerBuilder().
				WithName("apiserver").
				WithImage("k8s.gcr.io/kube-apiserver:v1.18.1").
				AddContainer().Build(),
			containerName: "scheduler",
			wantVersion:   semver.NewSemverOrDie("v1.18.1"),
			wantError:     false,
		},
		{
			name: "KAS rollout pending",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.18.1"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			compDeployment: test.NewDeploymentBuilder(test.NamespacedName{}).
				ContainerBuilder().
				WithName("scheduler").
				WithImage("k8s.gcr.io/scheduler:v1.17.2").
				AddContainer().Build(),
			kasDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: "apiserver", Namespace: "test"}).
				WithRolloutInProgress().
				ContainerBuilder().
				WithName("apiserver").
				WithImage("k8s.gcr.io/kube-apiserver:v1.18.1").
				AddContainer().Build(),
			containerName: "scheduler",
			wantVersion:   semver.NewSemverOrDie("v1.17.2"),
			wantError:     false,
		},
		{
			name: "Empty component deployment",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.18.1"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			compDeployment: test.NewDeploymentBuilder(test.NamespacedName{}).Build(),
			kasDeployment: test.NewDeploymentBuilder(test.NamespacedName{Name: "apiserver", Namespace: "test"}).
				WithReplicas(2).
				WithGeneration(2).
				WithStatus(appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: 2, AvailableReplicas: 2, UpdatedReplicas: 1}).
				ContainerBuilder().
				WithName("apiserver").
				WithImage("k8s.gcr.io/kube-apiserver:v1.18.1").
				AddContainer().Build(),
			containerName: "scheduler",
			wantVersion:   semver.NewSemverOrDie("v1.18.1"),
			wantError:     false,
		},
		{
			name: "KAS deployment missing",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.18.1"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			compDeployment: test.NewDeploymentBuilder(test.NamespacedName{}).Build(),
			wantVersion:    semver.NewSemverOrDie("v1.18.1"),
			wantError:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().Build()
			td := NewTemplateDataBuilder().
				WithCluster(tc.cluster).
				WithClient(cli).
				Build()
			if tc.kasDeployment != nil {
				cli.Create(context.TODO(), tc.kasDeployment)
			}
			ver, err := td.GetControlPlaneComponentVersion(tc.compDeployment, tc.containerName)
			if e, a := tc.wantError, err != nil; e != a {
				t.Logf("error: %v", err)
				t.Errorf("Want error %t, but got %t", e, a)
			}
			if e, a := tc.wantVersion != nil, ver != nil; e != a {
				t.Errorf("Want version %t, but got %t", e, a)
			}
			if e, a := tc.wantVersion, ver; e != nil && a != nil && !a.Equal(e) {
				t.Errorf("Want version %s, but got %s", e, a)
			}
		})
	}
}
