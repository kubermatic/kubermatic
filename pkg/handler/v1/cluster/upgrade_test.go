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

package cluster_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	k8csemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetClusterUpgrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                       string
		cluster                    *kubermaticv1.Cluster
		existingKubermaticObjs     []ctrlruntimeclient.Object
		existingMachineDeployments []*clusterv1alpha1.MachineDeployment
		apiUser                    apiv1.User
		versions                   []*version.Version
		updates                    []*version.Update
		wantUpdates                []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName}
				c.Spec.Version = *k8csemver.NewSemverOrDie("1.6.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			existingMachineDeployments: []*clusterv1alpha1.MachineDeployment{},
			apiUser:                    *test.GenDefaultAPIUser(),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
			},
		},
		{
			name: "upgrade available but restricted by kubelet versions",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName}
				c.Spec.Version = *k8csemver.NewSemverOrDie("1.6.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			existingMachineDeployments: func() []*clusterv1alpha1.MachineDeployment {
				mds := make([]*clusterv1alpha1.MachineDeployment, 0, 1)
				mdWithOldKubelet := test.GenTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false)
				mdWithOldKubelet.Spec.Template.Spec.Versions.Kubelet = "v1.4.0"
				mds = append(mds, mdWithOldKubelet)
				return mds
			}(),
			apiUser: *test.GenDefaultAPIUser(),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version:                    semver.MustParse("1.7.0"),
					RestrictedByKubeletVersion: true,
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
			},
		},
		{
			name: "no available",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName}
				c.Spec.Version = *k8csemver.NewSemverOrDie("1.6.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			existingMachineDeployments: []*clusterv1alpha1.MachineDeployment{},
			apiUser:                    *test.GenDefaultAPIUser(),
			wantUpdates:                []*apiv1.MasterVersion{},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{},
		},
		{
			name: "upgrade available for OpenShift",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName}
				c.Annotations = map[string]string{"kubermatic.io/openshift": "true"}
				c.Spec.Version = *k8csemver.NewSemverOrDie("5.1")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			existingMachineDeployments: []*clusterv1alpha1.MachineDeployment{},
			apiUser:                    *test.GenDefaultAPIUser(),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("5.2"),
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("5.1"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("5.2"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "5.1.0",
					To:        "5.2.0",
					Automatic: true,
					Type:      apiv1.OpenShiftClusterType,
				},
			},
		},
		{
			name: "upgrade not available for OpenShift (versions 3.11.*)",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName}
				c.Annotations = map[string]string{"kubermatic.io/openshift": "true"}
				c.Spec.Version = *k8csemver.NewSemverOrDie("3.11.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			existingMachineDeployments: []*clusterv1alpha1.MachineDeployment{},
			apiUser:                    *test.GenDefaultAPIUser(),
			wantUpdates:                []*apiv1.MasterVersion{},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11.0"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("3.11.1"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("3.11.2"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("3.11.3"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("5.2"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "3.11.0",
					To:        "3.11.*",
					Automatic: true,
					Type:      apiv1.OpenShiftClusterType,
				},
			},
		},
		{
			name: "the admin John can get available upgrades for Bob cluster",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now())
				c.Labels = map[string]string{"user": test.UserName, kubermaticv1.ProjectIDLabelKey: "my-first-project-ID"}
				c.Spec.Version = *k8csemver.NewSemverOrDie("1.6.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				genUser("John", "john@acme.com", true),
			),
			existingMachineDeployments: []*clusterv1alpha1.MachineDeployment{},
			apiUser:                    *test.GenAPIUser("John", "john@acme.com"),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
			},
		},
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/foo/upgrades", test.ProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{testStruct.cluster}
			kubermaticObj = append(kubermaticObj, testStruct.existingKubermaticObjs...)
			var machineObj []ctrlruntimeclient.Object
			for _, existingMachineDeployment := range testStruct.existingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}

			ep, _, err := test.CreateTestEndpointAndGetClients(testStruct.apiUser, nil, []ctrlruntimeclient.Object{}, machineObj, kubermaticObj, testStruct.versions, testStruct.updates, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create testStruct endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var gotUpdates []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotUpdates)
			if err != nil {
				t.Fatal(err)
			}

			test.CompareVersions(t, gotUpdates, testStruct.wantUpdates)
		})
	}
}

func TestUpgradeClusterNodeDeployments(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		Name                       string
		Body                       string
		HTTPStatus                 int
		ExpectedVersion            string
		ProjectIDToSync            string
		ClusterIDToSync            string
		ExistingProject            *kubermaticv1.Project
		ExistingKubermaticUser     *kubermaticv1.User
		ExistingAPIUser            *apiv1.User
		ExistingCluster            *kubermaticv1.Cluster
		ExistingMachineDeployments []ctrlruntimeclient.Object
		ExistingKubermaticObjs     []ctrlruntimeclient.Object
	}{
		{
			Name:            "scenario 1: upgrade node deployments",
			Body:            `{"version":"1.11.1"}`,
			HTTPStatus:      http.StatusOK,
			ExpectedVersion: "1.11.1",
			ClusterIDToSync: test.GenDefaultCluster().Name,
			ProjectIDToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenDefaultCluster()
					cluster.Spec.Version = *k8csemver.NewSemverOrDie("1.12.1")
					return cluster
				}(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []ctrlruntimeclient.Object{
				test.GenTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
				test.GenTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
			},
		},
		{
			Name:            "scenario 2: fail to upgrade node deployments",
			Body:            `{"version":"1.11.1"}`,
			HTTPStatus:      http.StatusBadRequest,
			ExpectedVersion: "v9.9.9",
			ClusterIDToSync: test.GenDefaultCluster().Name,
			ProjectIDToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenDefaultCluster()
					cluster.Spec.Version = *k8csemver.NewSemverOrDie("1.1.1")
					return cluster
				}(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []ctrlruntimeclient.Object{
				test.GenTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
				test.GenTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
			},
		},
		{
			Name:            "scenario 3: the admin John can upgrade Bob's node deployments",
			Body:            `{"version":"1.11.1"}`,
			HTTPStatus:      http.StatusOK,
			ExpectedVersion: "1.11.1",
			ClusterIDToSync: test.GenDefaultCluster().Name,
			ProjectIDToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				func() *kubermaticv1.Cluster {
					cluster := test.GenDefaultCluster()
					cluster.Spec.Version = *k8csemver.NewSemverOrDie("1.12.1")
					return cluster
				}(),
				genUser("John", "john@acme.com", true),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExistingMachineDeployments: []ctrlruntimeclient.Object{
				test.GenTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil, false),
				test.GenTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil, false),
			},
		},
	}

	for _, tc := range testcases {
		t.Logf("entering")
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/upgrades",
				tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			var machineObj []ctrlruntimeclient.Object
			var kubernetesObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			machineObj = append(machineObj, tc.ExistingMachineDeployments...)
			ep, cs, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			mds := &clusterv1alpha1.MachineDeploymentList{}
			if err := cs.FakeClient.List(context.TODO(), mds); err != nil {
				t.Fatalf("failed to list machine deployments: %v", err)
			}

			for _, md := range mds.Items {
				if md.Spec.Template.Spec.Versions.Kubelet != tc.ExpectedVersion {
					t.Fatalf("version %s does not match expected version %s: %v", md.Spec.Template.Spec.Versions.Kubelet, tc.ExpectedVersion, err)
				}
			}
		})
	}
}

func TestGetNodeUpgrades(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		controlPlaneVersion    string
		apiUser                apiv1.User
		existingUpdates        []*version.Update
		existingVersions       []*version.Version
		expectedOutput         []*apiv1.MasterVersion
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:                "only the same major version and no more than 2 minor versions behind the control plane",
			controlPlaneVersion: "1.6.0",
			apiUser:             *test.GenDefaultAPIUser(),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
			},
			existingUpdates: []*version.Update{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
				},
			},
			existingVersions: []*version.Version{
				{
					Version: semver.MustParse("0.0.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("0.1.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.0.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.4.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.4.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.5.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.5.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.6.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.7.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("2.0.0"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			expectedOutput: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.4.0"),
				},
				{
					Version: semver.MustParse("1.4.1"),
				},
				{
					Version: semver.MustParse("1.5.0"),
				},
				{
					Version: semver.MustParse("1.5.1"),
				},
			},
		},
		{
			name:                "only 2 minor versions for OpenShift cluster",
			controlPlaneVersion: "4.10.0&type=openshift",
			apiUser:             *test.GenDefaultAPIUser(),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
			},
			existingUpdates: []*version.Update{
				{
					From:      "4.10.0",
					To:        "4.10.1",
					Automatic: false,
					Type:      apiv1.OpenShiftClusterType,
				},
			},
			existingVersions: []*version.Version{
				{
					Version: semver.MustParse("1.7.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("2.0.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("4.10.0"),
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.10.1"),
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.11.0"),
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("5.10.0"),
					Type:    apiv1.OpenShiftClusterType,
				},
			},
			expectedOutput: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("4.10.0"),
				},
				{
					Version: semver.MustParse("4.10.1"),
				},
			},
		},
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/upgrades/node?control_plane_version=%s", testStruct.controlPlaneVersion), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(testStruct.apiUser, nil, testStruct.existingKubermaticObjs,
				testStruct.existingVersions, testStruct.existingUpdates, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create testStruct endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var response []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			test.CompareVersions(t, response, testStruct.expectedOutput)
		})
	}
}

func TestGetMasterVersionsEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		clusterType            string
		apiUser                apiv1.User
		existingUpdates        []*version.Update
		existingVersions       []*version.Version
		expectedOutput         []*apiv1.MasterVersion
		existingKubermaticObjs []ctrlruntimeclient.Object
	}{
		{
			name:        "get all OpenShift versions",
			clusterType: apiv1.OpenShiftClusterType,
			apiUser:     *test.GenDefaultAPIUser(),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
			},
			existingUpdates: []*version.Update{},
			existingVersions: []*version.Version{
				{
					Version: semver.MustParse("1.13.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
			expectedOutput: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("3.11"),
					Default: false,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
				},
			},
		},
		{
			name:        "get default only kubernetes versions",
			clusterType: "",
			apiUser:     *test.GenDefaultAPIUser(),
			existingKubermaticObjs: []ctrlruntimeclient.Object{
				test.GenTestSeed(),
				test.GenDefaultUser(),
			},
			existingUpdates: []*version.Update{},
			existingVersions: []*version.Version{
				{
					Version: semver.MustParse("1.13.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11.5"),
					Default: true,
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("3.11"),
					Default: false,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.1"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.2"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
				{
					Version: semver.MustParse("4.3"),
					Default: true,
					Type:    apiv1.OpenShiftClusterType,
				},
			},
			expectedOutput: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.13.5"),
					Default: true,
				},
				{
					Version: semver.MustParse("3.11.5"),
					Default: true,
				},
			},
		},
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			if len(testStruct.clusterType) > 0 {
				testStruct.clusterType = fmt.Sprintf("?type=%s", testStruct.clusterType)
			}
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/upgrades/cluster%s", testStruct.clusterType), nil)
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(testStruct.apiUser, nil, testStruct.existingKubermaticObjs,
				testStruct.existingVersions, testStruct.existingUpdates, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create testStruct endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var response []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			test.CompareVersions(t, response, testStruct.expectedOutput)
		})
	}
}
