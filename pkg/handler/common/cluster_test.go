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

package common

import (
	"reflect"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExternalCCMMigration(t *testing.T) {
	const (
		kubernetesVersionToTest = "1.21.0"
	)

	t.Parallel()
	testCases := []struct {
		Name           string
		Datacenter     *kubermaticv1.Datacenter
		Cluster        *kubermaticv1.Cluster
		ExpectedStatus apiv1.ExternalCCMMigrationStatus
	}{
		{
			Name: "scenario 1: CCM migration not needed since the beginning",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationNotNeeded,
		},
		{
			Name: "scenario 2: CCM migration not needed because cluster has already migrated",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
				Status: kubermaticv1.ClusterStatus{
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationNotNeeded,
		},
		{
			Name: "scenario 3: CCM migration supported",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationSupported,
		},
		{
			Name: "scenario 4: CCM migration unsupported",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{},
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationUnsupported,
		},
		{
			Name: "scenario 5: external CCM migration in progress, cluster condition existing",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
				Status: kubermaticv1.ClusterStatus{
					Conditions: []kubermaticv1.ClusterCondition{
						{
							Type:   kubermaticv1.ClusterConditionCSIKubeletMigrationCompleted,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationInProgress,
		},
		{
			Name: "scenario 6: external CCM migration in progress, cluster condition not existing",
			Datacenter: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{}},
			},
			Cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						kubermaticv1.CCMMigrationNeededAnnotation: "",
						kubermaticv1.CSIMigrationNeededAnnotation: "",
					},
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureExternalCloudProvider: true,
					},
					Version: *semver.NewSemverOrDie(kubernetesVersionToTest),
				},
			},
			ExpectedStatus: apiv1.ExternalCCMMigrationInProgress,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ccmStatus := convertInternalCCMStatusToExternal(tc.Cluster, tc.Datacenter)
			if !reflect.DeepEqual(ccmStatus, tc.ExpectedStatus) {
				t.Fatalf("Received status %v, expected status: %v", ccmStatus, tc.ExpectedStatus)
			}
		})
	}
}
