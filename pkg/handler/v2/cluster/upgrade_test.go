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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	k8csemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetClusterUpgrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                       string
		cluster                    *kubermaticv1.Cluster
		existingKubermaticObjs     []runtime.Object
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
			existingKubermaticObjs:     test.GenDefaultKubermaticObjects(),
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
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
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
			existingKubermaticObjs:     test.GenDefaultKubermaticObjects(),
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
			existingKubermaticObjs:     test.GenDefaultKubermaticObjects(),
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
			existingKubermaticObjs:     test.GenDefaultKubermaticObjects(),
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
			existingKubermaticObjs:     test.GenDefaultKubermaticObjects(genUser("John", "john@acme.com", true)),
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
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/projects/%s/clusters/foo/upgrades", test.ProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{testStruct.cluster}
			kubermaticObj = append(kubermaticObj, testStruct.existingKubermaticObjs...)
			var machineObj []runtime.Object
			for _, existingMachineDeployment := range testStruct.existingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}

			ep, _, err := test.CreateTestEndpointAndGetClients(testStruct.apiUser, nil, []runtime.Object{}, machineObj, kubermaticObj, testStruct.versions, testStruct.updates, hack.NewTestRouting)
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
