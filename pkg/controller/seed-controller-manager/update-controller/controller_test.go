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

package updatecontroller

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetOldestAvailableVersion(t *testing.T) {
	makeSet := func(replicas int32, version string) appsv1.ReplicaSet {
		return appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					resources.VersionLabel: version,
				},
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas: replicas,
			},
		}
	}

	testcases := []struct {
		sets     []appsv1.ReplicaSet
		expected *semver.Semver
	}{
		{
			sets: []appsv1.ReplicaSet{
				makeSet(0, ""),
			},
			expected: nil,
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(1, ""),
			},
			expected: nil,
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(1, "not-a-version"),
			},
			expected: nil,
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(0, "1.2.3"),
			},
			expected: nil,
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(1, "1.2.3"),
			},
			expected: semver.NewSemverOrDie("1.2.3"),
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(1, "1.2.3"),
				makeSet(1, "1.2.4"),
			},
			expected: semver.NewSemverOrDie("1.2.3"),
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(0, "1.2.3"),
				makeSet(1, "1.2.4"),
			},
			expected: semver.NewSemverOrDie("1.2.4"),
		},
		{
			sets: []appsv1.ReplicaSet{
				makeSet(1, "1.2.4"),
				makeSet(0, "1.2.3"),
			},
			expected: semver.NewSemverOrDie("1.2.4"),
		},
	}

	for i, tt := range testcases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			result := getOldestAvailableVersion(zap.NewNop().Sugar(), tt.sets)
			if tt.expected == nil {
				if result != nil {
					t.Fatalf("Expected nil, but got %s.", result.String())
				}
			} else if !result.Equal(tt.expected) {
				t.Fatalf("Expected %s, but got %v.", tt.expected.String(), result)
			}
		})
	}
}

func TestHasOwnerRefToAny(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Test",
					Name: "test",
				},
				{
					Kind: "Secret",
					Name: "test",
				},
				{
					Kind: "Test",
					Name: "another-owner",
				},
			},
		},
	}

	testcases := []struct {
		kind     string
		names    sets.Set[string]
		expected bool
	}{
		{
			kind:     "Test",
			names:    sets.New("test"),
			expected: true,
		},
		{
			kind:     "Test",
			names:    sets.New("another-owner"),
			expected: true,
		},
		{
			kind:     "Secret",
			names:    sets.New("test"),
			expected: true,
		},
		{
			kind:     "Test",
			names:    sets.New("non-existent"),
			expected: false,
		},
		{
			kind:     "Deployment",
			names:    sets.New("non-existent"),
			expected: false,
		},
	}

	for i, tt := range testcases {
		t.Run(fmt.Sprintf("testcase %d", i), func(t *testing.T) {
			result := hasOwnerRefToAny(cm, tt.kind, tt.names)
			if result != tt.expected {
				t.Fatalf("Expected %v, but got the opposite.", tt.expected)
			}
		})
	}
}

func TestGetNextApiServerVersion(t *testing.T) {
	// This test (and this controller, really) is not about unit testing
	// all the possible incompatibility options, so we skip them here.
	versions := kubermaticv1.KubermaticVersioningConfiguration{
		Versions: []semver.Semver{
			*semver.NewSemverOrDie("1.20.0"),
			*semver.NewSemverOrDie("1.20.1"),
			*semver.NewSemverOrDie("1.20.2"),
			*semver.NewSemverOrDie("1.21.0"),
			*semver.NewSemverOrDie("1.21.3"),
			*semver.NewSemverOrDie("1.22.5"),
			*semver.NewSemverOrDie("1.23.0"),
			*semver.NewSemverOrDie("1.24.0"),
		},
		Updates: []kubermaticv1.Update{
			{
				From: "1.20.*",
				To:   "1.20.*",
			},
			{
				From: "1.20.*",
				To:   "1.21.*",
			},
			{
				From: "1.21.*",
				To:   "1.21.*",
			},
			{
				From: "1.21.*",
				To:   "1.22.*",
			},
			{
				From: "1.22.*",
				To:   "1.22.*",
			},
			{
				From: "1.22.*",
				To:   "1.23.*",
			},
			{
				From: "1.23.*",
				To:   "1.23.*",
			},
			// no rule for 1.23 => 1.24!
			{
				From: "1.24.*",
				To:   "1.24.*",
			},
		},
	}

	testcases := []struct {
		name             string
		specVersion      semver.Semver
		apiserverVersion semver.Semver
		expected         *semver.Semver
		expectedErr      bool
	}{
		{
			name:             "allow simple patch release update",
			apiserverVersion: *semver.NewSemverOrDie("1.20.0"),
			specVersion:      *semver.NewSemverOrDie("1.20.1"),
			expected:         semver.NewSemverOrDie("1.20.1"),
		},
		{
			name:             "allow updating to the next minor",
			apiserverVersion: *semver.NewSemverOrDie("1.20.2"),
			specVersion:      *semver.NewSemverOrDie("1.21.0"),
			expected:         semver.NewSemverOrDie("1.21.0"),
		},
		{
			name:             "allow updating to the max next minor",
			apiserverVersion: *semver.NewSemverOrDie("1.20.2"),
			specVersion:      *semver.NewSemverOrDie("1.21.3"),
			expected:         semver.NewSemverOrDie("1.21.3"),
		},
		{
			name:             "take the highest patch release of the next minor when upgrading across multiple minors",
			apiserverVersion: *semver.NewSemverOrDie("1.20.2"),
			specVersion:      *semver.NewSemverOrDie("1.23.0"),
			expected:         semver.NewSemverOrDie("1.21.3"),
		},
		{
			name:             "currently active version is not configured anymore",
			apiserverVersion: *semver.NewSemverOrDie("1.20.7"),
			specVersion:      *semver.NewSemverOrDie("1.21.0"),
			expected:         semver.NewSemverOrDie("1.21.0"),
		},
		{
			name:             "no path configured",
			apiserverVersion: *semver.NewSemverOrDie("1.23.0"),
			specVersion:      *semver.NewSemverOrDie("1.24.0"),
			expectedErr:      true,
		},
		{
			name:             "unsupported target version",
			apiserverVersion: *semver.NewSemverOrDie("1.20.0"),
			specVersion:      *semver.NewSemverOrDie("1.20.9"),
			expectedErr:      true,
		},
		{
			name:             "unsupported target version across a minor release",
			apiserverVersion: *semver.NewSemverOrDie("1.21.3"),
			specVersion:      *semver.NewSemverOrDie("1.22.0"),
			expectedErr:      true,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: tt.specVersion,
					Cloud: kubermaticv1.CloudSpec{
						ProviderName: string(kubermaticv1.AWSCloudProvider),
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: kubermaticv1.ClusterVersionsStatus{
						Apiserver: tt.apiserverVersion,
					},
				},
			}

			config := &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Versions: versions,
				},
			}

			nextVersion, err := getNextApiserverVersion(context.Background(), config, cluster)
			if err != nil {
				if !tt.expectedErr {
					t.Fatalf("Expected next version %s, but got error: %v", tt.expected.String(), err)
				}
			} else {
				if tt.expectedErr {
					t.Fatalf("Expected error, but got result instead: %s", nextVersion.String())
				}

				if !nextVersion.Equal(tt.expected) {
					t.Fatalf("Expected %s, but got %s instead.", tt.expected.String(), nextVersion.String())
				}
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	versions := kubermaticv1.KubermaticVersioningConfiguration{
		Versions: []semver.Semver{
			*semver.NewSemverOrDie("1.20.0"),
			*semver.NewSemverOrDie("1.20.1"),
			*semver.NewSemverOrDie("1.20.2"),
			*semver.NewSemverOrDie("1.21.0"),
			*semver.NewSemverOrDie("1.21.3"),
			*semver.NewSemverOrDie("1.22.5"),
			*semver.NewSemverOrDie("1.23.0"),
			*semver.NewSemverOrDie("1.24.0"),
		},
		Updates: []kubermaticv1.Update{
			{
				From: "1.20.*",
				To:   "1.20.*",
			},
			{
				From: "1.20.*",
				To:   "1.21.*",
			},
			{
				From: "1.21.*",
				To:   "1.21.*",
			},
			{
				From: "1.21.*",
				To:   "1.22.*",
			},
			{
				From: "1.22.*",
				To:   "1.22.*",
			},
			{
				From: "1.22.*",
				To:   "1.23.*",
			},
			{
				From: "1.23.*",
				To:   "1.23.*",
			},
			// no rule for 1.23 => 1.24!
			{
				From: "1.24.*",
				To:   "1.24.*",
			},
		},
	}

	testcases := []struct {
		name           string
		specVersion    semver.Semver
		clusterStatus  kubermaticv1.ClusterVersionsStatus
		currentStatus  controlPlaneStatus
		healthy        bool
		expectedStatus kubermaticv1.ClusterVersionsStatus
		expectedErr    bool
	}{
		// ///////////////////////////////////////////////////////
		// all of the following tests ignore the existence of nodes;
		// the controller will just continue updating the control plane
		// if the cluster has no nodes (currentStatus.nodes being nil)

		{
			name:          "new cluster created, set the initial status",
			specVersion:   *semver.NewSemverOrDie("1.23.0"),
			healthy:       false,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.23.0"),
				Apiserver:         *semver.NewSemverOrDie("1.23.0"),
				ControllerManager: *semver.NewSemverOrDie("1.23.0"),
				Scheduler:         *semver.NewSemverOrDie("1.23.0"),
			},
		},
		{
			name:        "cluster is healthy and up-to-date, nothing to do",
			specVersion: *semver.NewSemverOrDie("1.20.1"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.20.1"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.20.1"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.20.1"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
		},
		{
			name:        "cluster up-to-date, but not yet healthy, do nothing and wait for it to be come healthy",
			specVersion: *semver.NewSemverOrDie("1.20.1"),
			healthy:     false,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.20.1"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.20.1"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.20.1"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
		},
		{
			name:        "vanilla cluster that was just told to be updated by changing its spec",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.20.1"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.20.1"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"), // this should be updated to the spec
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
		},
		{
			name:        "waiting for the new apiserver to become healthy before updating the controlplanne version, i.e. do nothing yet",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     false,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.20.1"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
		},
		{
			name:        "apiserver is updated, but control plane is still rolling out (e.g. etcd is still updating), do nothing and wait for healthy",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     false,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
		},
		{
			name:        "apiserver became healthy, mark control plane as updated and mark cm/scheduler to be updated",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.20.1"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.20.1"),
				Scheduler:         *semver.NewSemverOrDie("1.20.1"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},
		{
			name:        "wait for control plane to come up, do nothing",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     false,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.20.1"),
				scheduler:         semver.NewSemverOrDie("1.20.1"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},
		{
			name:        "update completed, back to a vanilla cluster",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.21.0"),
				scheduler:         semver.NewSemverOrDie("1.21.0"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},
		{
			name:        "update to 1.23 and ensure we jump to 1.22",
			specVersion: *semver.NewSemverOrDie("1.23.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.21.0"),
				scheduler:         semver.NewSemverOrDie("1.21.0"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.22.5"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},

		// ///////////////////////////////////////////////////////
		// the following tests demonstrate how we must wait for nodes
		// before proceeding with the control plane

		{
			name:        "cluster still has ancient nodes, cannot progress with updates",
			specVersion: *semver.NewSemverOrDie("1.23.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.22.0"),
				Apiserver:         *semver.NewSemverOrDie("1.22.0"),
				ControllerManager: *semver.NewSemverOrDie("1.22.0"),
				Scheduler:         *semver.NewSemverOrDie("1.22.0"),
				OldestNodeVersion: semver.NewSemverOrDie("1.20.0"), // allows max 1.22.* control plane
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.22.0"),
				controllerManager: semver.NewSemverOrDie("1.22.0"),
				scheduler:         semver.NewSemverOrDie("1.22.0"),
				nodes:             semver.NewSemverOrDie("1.20.0"), // allows max 1.22.* control plane
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.22.0"),
				Apiserver:         *semver.NewSemverOrDie("1.22.0"),
				ControllerManager: *semver.NewSemverOrDie("1.22.0"),
				Scheduler:         *semver.NewSemverOrDie("1.22.0"),
			},
		},

		{
			name:        "cluster still has ancient nodes, but is (erroneously) on its way to update and so we should allow this to complete",
			specVersion: *semver.NewSemverOrDie("1.23.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.22.0"),
				Apiserver:         *semver.NewSemverOrDie("1.23.0"),
				ControllerManager: *semver.NewSemverOrDie("1.22.0"),
				Scheduler:         *semver.NewSemverOrDie("1.22.0"),
				OldestNodeVersion: semver.NewSemverOrDie("1.20.0"), // allows max 1.22.* control plane
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.23.0"),
				controllerManager: semver.NewSemverOrDie("1.22.0"),
				scheduler:         semver.NewSemverOrDie("1.22.0"),
				nodes:             semver.NewSemverOrDie("1.20.0"), // allows max 1.22.* control plane
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.23.0"),
				Apiserver:         *semver.NewSemverOrDie("1.23.0"),
				ControllerManager: *semver.NewSemverOrDie("1.23.0"), // allow them to follow the apiserver's version
				Scheduler:         *semver.NewSemverOrDie("1.23.0"), // allow them to follow the apiserver's version
			},
		},

		{
			name:        "cluster still has old nodes, but young enough to permit an update",
			specVersion: *semver.NewSemverOrDie("1.23.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.22.0"),
				Apiserver:         *semver.NewSemverOrDie("1.22.0"),
				ControllerManager: *semver.NewSemverOrDie("1.22.0"),
				Scheduler:         *semver.NewSemverOrDie("1.22.0"),
				OldestNodeVersion: semver.NewSemverOrDie("1.21.0"), // allows max 1.23.* control plane
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.22.0"),
				controllerManager: semver.NewSemverOrDie("1.22.0"),
				scheduler:         semver.NewSemverOrDie("1.22.0"),
				nodes:             semver.NewSemverOrDie("1.21.0"), // allows max 1.23.* control plane
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.22.0"),
				Apiserver:         *semver.NewSemverOrDie("1.23.0"), // updated!
				ControllerManager: *semver.NewSemverOrDie("1.22.0"),
				Scheduler:         *semver.NewSemverOrDie("1.22.0"),
			},
		},

		// //////////////////////////////////////////////////
		// tests for downgrading a cluster (same minor, different patch release)

		{
			name:        "vanilla cluster that was just told to be downgraded by changing its spec",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.3"),
				Apiserver:         *semver.NewSemverOrDie("1.21.3"),
				ControllerManager: *semver.NewSemverOrDie("1.21.3"),
				Scheduler:         *semver.NewSemverOrDie("1.21.3"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.3"),
				controllerManager: semver.NewSemverOrDie("1.21.3"),
				scheduler:         semver.NewSemverOrDie("1.21.3"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.3"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"), // this should be updated to the spec
				ControllerManager: *semver.NewSemverOrDie("1.21.3"),
				Scheduler:         *semver.NewSemverOrDie("1.21.3"),
			},
		},
		{
			name:        "downgraded apiserver came alive, rest of control plane should downgrade as well",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.3"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.3"),
				Scheduler:         *semver.NewSemverOrDie("1.21.3"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.21.3"),
				scheduler:         semver.NewSemverOrDie("1.21.3"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},
		{
			name:        "downgrade completes",
			specVersion: *semver.NewSemverOrDie("1.21.0"),
			healthy:     true,
			clusterStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
			currentStatus: controlPlaneStatus{
				apiserver:         semver.NewSemverOrDie("1.21.0"),
				controllerManager: semver.NewSemverOrDie("1.21.0"),
				scheduler:         semver.NewSemverOrDie("1.21.0"),
			},
			expectedStatus: kubermaticv1.ClusterVersionsStatus{
				ControlPlane:      *semver.NewSemverOrDie("1.21.0"),
				Apiserver:         *semver.NewSemverOrDie("1.21.0"),
				ControllerManager: *semver.NewSemverOrDie("1.21.0"),
				Scheduler:         *semver.NewSemverOrDie("1.21.0"),
			},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testcluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					Version: tt.specVersion,
					Cloud: kubermaticv1.CloudSpec{
						ProviderName: string(kubermaticv1.AWSCloudProvider),
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: tt.clusterStatus,
				},
			}

			if tt.healthy {
				cluster.Status.ExtendedHealth = kubermaticv1.ExtendedClusterHealth{
					Apiserver:                    kubermaticv1.HealthStatusUp,
					ApplicationController:        kubermaticv1.HealthStatusUp,
					Scheduler:                    kubermaticv1.HealthStatusUp,
					Controller:                   kubermaticv1.HealthStatusUp,
					MachineController:            kubermaticv1.HealthStatusUp,
					Etcd:                         kubermaticv1.HealthStatusUp,
					OpenVPN:                      kubermaticv1.HealthStatusUp,
					CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
					UserClusterControllerManager: kubermaticv1.HealthStatusUp,
				}
			}

			config := &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Versions: versions,
				},
			}

			configGetter, err := kubernetesprovider.StaticKubermaticConfigurationGetterFactory(config)
			if err != nil {
				t.Fatalf("Failed to create config getter: %v", err)
			}

			rec := &Reconciler{
				Client:       fake.NewClientBuilder().WithObjects(cluster).Build(),
				configGetter: configGetter,
				log:          zap.NewNop().Sugar(),
				versions:     kubermatic.GetFakeVersions(),
				recorder:     events.NewFakeRecorder(10),
				cpChecker: func(_ context.Context, _ ctrlruntimeclient.Client, _ *zap.SugaredLogger, _ *kubermaticv1.Cluster) (*controlPlaneStatus, error) {
					return &tt.currentStatus, nil
				},
			}

			err = rec.reconcile(context.Background(), rec.log, cluster)
			if err != nil {
				if !tt.expectedErr {
					t.Fatalf("Got unexpected error: %v", err)
				}
			} else {
				if tt.expectedErr {
					t.Fatalf("Expected error, but got none.")
				}

				newCluster := &kubermaticv1.Cluster{}
				err := rec.Get(context.Background(), ctrlruntimeclient.ObjectKeyFromObject(cluster), newCluster)
				if err != nil {
					t.Fatalf("Failed to find cluster: %v", err)
				}

				if !tt.expectedStatus.Apiserver.Equal(&newCluster.Status.Versions.Apiserver) {
					t.Errorf("Expected apiserver to be %v, but is %v.", tt.expectedStatus.Apiserver, newCluster.Status.Versions.Apiserver)
				}

				if !tt.expectedStatus.ControllerManager.Equal(&newCluster.Status.Versions.ControllerManager) {
					t.Errorf("Expected controller-manager to be %v, but is %v.", tt.expectedStatus.ControllerManager, newCluster.Status.Versions.ControllerManager)
				}

				if !tt.expectedStatus.Scheduler.Equal(&newCluster.Status.Versions.Scheduler) {
					t.Errorf("Expected scheduler to be %v, but is %v.", tt.expectedStatus.Scheduler, newCluster.Status.Versions.Scheduler)
				}
			}
		})
	}
}
