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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Version: *ksemver.NewSemverOrDie("v1.20.4"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.String{},
		},
		{
			name: "CSI migration feature disabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cluster-a",
					Annotations: map[string]string{},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						kubermaticv1.ClusterFeatureCSIMigration:          false,
					},
					Version: *ksemver.NewSemverOrDie("v1.20.4"),
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
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						kubermaticv1.ClusterFeatureCSIMigration:          true,
					},
					Version: *ksemver.NewSemverOrDie("v1.20.4"),
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
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						kubermaticv1.ClusterFeatureCSIMigration:          true,
					},
					Version: *ksemver.NewSemverOrDie("v1.20.4"),
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
		{
			name: "CSI migration completed on 1.21+",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						kubermaticv1.ClusterFeatureCSIMigration:          true,
					},
					Version: *ksemver.NewSemverOrDie("v1.21.0"),
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
			wantFeatureGates: sets.NewString("CSIMigration=true", "CSIMigrationOpenStack=true", "ExpandCSIVolumes=true", "InTreePluginOpenStackUnregister=true"),
		},
		{
			name: "CSI migration on non-OpenStack provider",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster-a",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						AWS: &kubermaticv1.AWSCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
						kubermaticv1.ClusterFeatureCSIMigration:          true,
					},
					Version: *ksemver.NewSemverOrDie("v1.20.4"),
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "test",
				},
			},
			wantFeatureGates: sets.String{},
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
