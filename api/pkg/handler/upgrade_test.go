package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	ksemver "github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func TestGetClusterUpgradesV1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		cluster                *kubermaticv1.Cluster
		existingKubermaticObjs []runtime.Object
		apiUser                apiv1.User
		versions               []*version.MasterVersion
		updates                []*version.MasterUpdate
		wantUpdates            []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": test.UserName},
				},
				Spec: kubermaticv1.ClusterSpec{Version: *ksemver.NewSemverOrDie("1.6.0")},
			},
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			apiUser:                *test.GenDefaultAPIUser(),
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			updates: []*version.MasterUpdate{
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
		},
		{
			name: "no available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": test.UserName},
				},
				Spec: kubermaticv1.ClusterSpec{Version: *ksemver.NewSemverOrDie("1.6.0")},
			},
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			apiUser:                *test.GenDefaultAPIUser(),
			wantUpdates:            []*apiv1.MasterVersion{},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
			},
			updates: []*version.MasterUpdate{},
		},
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/foo/upgrades", test.ProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{testStruct.cluster}
			kubermaticObj = append(kubermaticObj, testStruct.existingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(testStruct.apiUser, []runtime.Object{}, kubermaticObj, testStruct.versions, testStruct.updates, hack.NewTestRouting)
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

			if diff := deep.Equal(gotUpdates, testStruct.wantUpdates); diff != nil {
				t.Fatalf("got different upgrade response than expected. Diff: %v", diff)
			}
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
		ExistingMachineDeployments []*clusterv1alpha1.MachineDeployment
		ExistingKubermaticObjs     []runtime.Object
	}{
		// scenario 1
		{
			Name:                   "scenario 1: upgrade node deployments",
			Body:                   `{"version":"1.11.1"}`,
			HTTPStatus:             http.StatusOK,
			ExpectedVersion:        "1.11.1",
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingMachineDeployments: []*clusterv1alpha1.MachineDeployment{
				genTestMachineDeployment("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":true}}`, nil),
				genTestMachineDeployment("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, nil),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/nodes/upgrades",
				tc.ProjectIDToSync, tc.ClusterIDToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			machineObj := []runtime.Object{}
			kubernetesObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			for _, existingMachineDeployment := range tc.ExistingMachineDeployments {
				machineObj = append(machineObj, existingMachineDeployment)
			}
			ep, cs, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			mds, err := cs.FakeMachineClient.ClusterV1alpha1().MachineDeployments(metav1.NamespaceSystem).List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("failed to list machine deployments: %v", err)
			}

			for _, md := range mds.Items {
				if md.Spec.Template.Spec.Versions.Kubelet != tc.ExpectedVersion {
					t.Fatalf("version does not match expected one: %v", err)
				}
			}
		})
	}
}

func TestGetClusterNodeUpgrades(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		cluster                *kubermaticv1.Cluster
		apiUser                apiv1.User
		existingKubermaticObjs []runtime.Object

		existingUpdates  []*version.MasterUpdate
		existingVersions []*version.MasterVersion
		expectedOutput   []*apiv1.MasterVersion
	}{
		{
			name: "only the same major version and no more than 2 minor versions behind the control plane",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": test.UserName},
				},
				Spec: kubermaticv1.ClusterSpec{Version: *ksemver.NewSemverOrDie("1.6.0")},
			},
			apiUser:                *test.GenDefaultAPIUser(),
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			existingUpdates: []*version.MasterUpdate{
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
			existingVersions: []*version.MasterVersion{
				{
					Version: semver.MustParse("0.0.1"),
				},
				{
					Version: semver.MustParse("0.1.0"),
				},
				{
					Version: semver.MustParse("1.0.0"),
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
				{
					Version: semver.MustParse("1.6.0"),
				},
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
				{
					Version: semver.MustParse("1.7.1"),
				},
				{
					Version: semver.MustParse("2.0.0"),
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
	}
	for _, testStruct := range tests {
		t.Run(testStruct.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/foo/nodes/upgrades", test.ProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{testStruct.cluster}
			kubermaticObj = append(kubermaticObj, testStruct.existingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(testStruct.apiUser, []runtime.Object{}, kubermaticObj,
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

			if diff := deep.Equal(response, testStruct.expectedOutput); diff != nil {
				t.Fatalf("got different versions response than expected. Diff: %v", diff)
			}
		})
	}
}
