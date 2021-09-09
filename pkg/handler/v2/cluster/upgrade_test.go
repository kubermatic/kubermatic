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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	k8csemver "k8c.io/kubermatic/v2/pkg/semver"

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
		versions                   []*semver.Version
		updates                    []operatorv1alpha1.Update
		incompatibilities          []operatorv1alpha1.Incompatibility
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
			versions: []*semver.Version{
				semver.MustParse("1.6.0"),
				semver.MustParse("1.6.1"),
				semver.MustParse("1.7.0"),
			},
			updates: []operatorv1alpha1.Update{
				{
					From: "1.6.0",
					To:   "1.6.1",
				},
				{
					From: "1.6.x",
					To:   "1.7.0",
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
			versions: []*semver.Version{
				semver.MustParse("1.6.0"),
				semver.MustParse("1.6.1"),
				semver.MustParse("1.7.0"),
			},
			updates: []operatorv1alpha1.Update{
				{
					From: "1.6.0",
					To:   "1.6.1",
				},
				{
					From: "1.6.x",
					To:   "1.7.0",
				},
			},
		},
		{
			name: "upgrade available but incompatible with the given provider",
			cluster: func() *kubermaticv1.Cluster {
				c := test.GenCluster("foo", "foo", "project", time.Now(), func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.Cloud.VSphere = &kubermaticv1.VSphereCloudSpec{}
				})
				c.Labels = map[string]string{"user": test.UserName}
				c.Spec.Version = *k8csemver.NewSemverOrDie("1.21.0")
				return c
			}(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			apiUser: *test.GenDefaultAPIUser(),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.21.1"),
				},
			},
			versions: []*semver.Version{
				semver.MustParse("1.21.0"),
				semver.MustParse("1.21.1"),
				semver.MustParse("1.22.0"),
				semver.MustParse("1.22.1"),
			},
			updates: []operatorv1alpha1.Update{
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
			},
			incompatibilities: []operatorv1alpha1.Incompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Condition: operatorv1alpha1.AlwaysCondition,
					Operation: operatorv1alpha1.UpdateOperation,
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
			versions: []*semver.Version{
				semver.MustParse("1.6.0"),
			},
			updates: []operatorv1alpha1.Update{},
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
			versions: []*semver.Version{
				semver.MustParse("1.6.0"),
				semver.MustParse("1.6.1"),
				semver.MustParse("1.7.0"),
			},
			updates: []operatorv1alpha1.Update{
				{
					From: "1.6.0",
					To:   "1.6.1",
				},
				{
					From: "1.6.x",
					To:   "1.7.0",
				},
			},
		},
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			dummyKubermaticConfiguration := operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: test.KubermaticNamespace,
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Versions: operatorv1alpha1.KubermaticVersionsConfiguration{
						Kubernetes: operatorv1alpha1.KubermaticVersioningConfiguration{
							Versions:                  testStruct.versions,
							Updates:                   testStruct.updates,
							ProviderIncompatibilities: testStruct.incompatibilities,
						},
					},
				},
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/foo/upgrades", test.ProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{testStruct.cluster}
			kubermaticObj = append(kubermaticObj, testStruct.existingKubermaticObjs...)
			var machineObj []ctrlruntimeclient.Object
			for _, existingMachineDeployment := range testStruct.existingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}

			ep, _, err := test.CreateTestEndpointAndGetClients(testStruct.apiUser, nil, []ctrlruntimeclient.Object{}, machineObj, kubermaticObj, &dummyKubermaticConfiguration, hack.NewTestRouting)
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
			), ExistingAPIUser: test.GenDefaultAPIUser(),
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
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v2/projects/%s/clusters/%s/nodes/upgrades",
				tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []ctrlruntimeclient.Object
			var machineObj []ctrlruntimeclient.Object
			var kubernetesObj []ctrlruntimeclient.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			machineObj = append(machineObj, tc.ExistingMachineDeployments...)
			ep, cs, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, hack.NewTestRouting)
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
