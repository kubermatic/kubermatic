package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-test/deep"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
)

func TestRemoveSensitiveDataFromCluster(t *testing.T) {
	t.Parallel()
	genClusterResource := func() *kubermaticv1.Cluster {
		return &kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "abcd",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Project",
						UID:        "",
						Name:       "myProjectInternalName",
					},
				},
			},
		}
	}
	genClusterWithAdminToken := func() *kubermaticv1.Cluster {
		cluster := genClusterResource()
		cluster.Address.AdminToken = "hfzj6l.w7hgc65nq9z4fxvl"
		cluster.Address.ExternalName = "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io"
		cluster.Address.IP = "35.194.142.199"
		cluster.Address.URL = "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"
		return cluster
	}
	genClusterWithAWS := func() *kubermaticv1.Cluster {
		cluster := genClusterResource()
		cluster.Spec.Cloud = &kubermaticv1.CloudSpec{
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:      "secretKeyID",
				SecretAccessKey:  "secreatAccessKey",
				SecurityGroupID:  "secuirtyGroupID",
				AvailabilityZone: "availablityZone",
			},
		}

		return cluster
	}
	scenarios := []struct {
		Name            string
		ExistingCluster *kubermaticv1.Cluster
		ExpectedCluster *kubermaticv1.Cluster
	}{
		{
			Name:            "scenaio 1: removes the admin token",
			ExistingCluster: genClusterWithAdminToken(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithAdminToken()
				cluster.Address.AdminToken = ""
				return cluster
			}(),
		},
		{
			Name:            "scenario 2: removes AWS cloud provider secrets",
			ExistingCluster: genClusterWithAWS(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithAWS()
				cluster.Spec.Cloud.AWS.AccessKeyID = ""
				cluster.Spec.Cloud.AWS.SecretAccessKey = ""
				return cluster
			}(),
		},
	}
	for _, tc := range scenarios {
		t.Run(tc.Name, func(t *testing.T) {
			actualCluster := removeSensitiveDataFromCluster(tc.ExistingCluster)
			if !equality.Semantic.DeepEqual(actualCluster, tc.ExpectedCluster) {
				t.Fatalf("%v", diff.ObjectDiff(tc.ExpectedCluster, actualCluster))
			}
		})
	}
}

func TestAssignSSHKeysToClusterEndpoint(t *testing.T) {
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKey         *kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:             "scenario 1: an ssh key that belongs to the given project is assigned to the cluster",
			Body:             `{"keys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `null`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: "john@acme.com",
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: "john@acme.com",
			},
			ExistingSSHKey: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "myProjectInternalName",
						},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "myProjectInternalName",
						},
					},
				},
			},
		},
		// scenario 2
		{
			Name:             "scenario 2: an ssh key that does not belong to the given project cannot be assigned to the cluster",
			Body:             `{"keys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":500,"message":"the given ssh key key-c08aa5c7abf34504f18552846485267d-yafn does not belong to the given project my-first-project (myProjectInternalName)"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "John",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: "john@acme.com",
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: "john@acme.com",
			},
			ExistingSSHKey: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "differentProject",
						},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "myProjectInternalName",
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/abcd/sshkeys", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			if tc.ExistingSSHKey != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingSSHKey)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateClusterEndpoint(t *testing.T) {
	testcases := []struct {
		Name                               string
		Body                               string
		ExpectedResponse                   string
		HTTPStatus                         int
		ExistingProject                    *kubermaticv1.Project
		ExistingKubermaticUser             *kubermaticv1.User
		ExistingAPIUser                    *apiv1.User
		ExistingSSHKey                     *kubermaticv1.UserSSHKey
		RewriteClusterNameAndNamespaceName bool
	}{
		// scenario 1
		{
			Name:             "scenario 1: a cluster with invalid spec is rejected",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","version":"","pause":false,"cloud":{"digitalocean":{"token":"dummy_token"},"dc":"do-fra1"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec \"Version\" is required but was not specified"}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: "john@acme.com",
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: "john@acme.com",
			},
		},
		// scenario 2
		{
			Name:                               "scenario 2: cluster is created when valid spec and ssh key are passed",
			Body:                               `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"fake":{"token":"dummy_token"},"dc":"do-fra1"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse:                   `{"metadata":{"name":"%s","creationTimestamp":null,"labels":{"worker-name":""},"ownerReferences":[{"apiVersion":"kubermatic.k8s.io/v1","kind":"Project","name":"myProjectInternalName","uid":""}]},"spec":{"cloud":{"dc":"do-fra1","fake":{"token":"dummy_token"}},"clusterNetwork":{"services":{"cidrBlocks":null},"pods":{"cidrBlocks":null},"dnsDomain":""},"version":"1.9.7","masterVersion":"","humanReadableName":"keen-snyder","workerName":"","pause":false},"address":{"url":"","externalName":"","kubeletToken":"","adminToken":"","ip":""},"status":{"lastUpdated":null,"health":{"apiserver":false,"scheduler":false,"controller":false,"machineController":false,"etcd":false,"lastTransitionTime":null},"lastDeployedMasterVersion":"","rootCA":{"key":"","cert":""},"apiserverCert":{"key":"","cert":""},"kubeletCert":{"key":"","cert":""},"apiserverSshKey":{"privateKey":"","publicKey":""},"serviceAccountKey":"","namespaceName":"%s","userName":"","userEmail":"john@acme.com"}}`,
			RewriteClusterNameAndNamespaceName: true,
			HTTPStatus:                         http.StatusCreated,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: "john@acme.com",
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: "john@acme.com",
			},
			ExistingSSHKey: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       "myProjectInternalName",
						},
					},
				},
			},
		},
		// scenario 3
		{
			Name:             "scenario 3: unable to create a cluster when the user doesn't belong to the project",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"digitalocean":{"token":"dummy_token"},"dc":"do-fra1"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: The user doesn't belong to the given project = myProjectInternalName"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: "john@acme.com",
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-secretProject",
							Name:  "secretProject",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: "john@acme.com",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects/myProjectInternalName/dc/us-central1/clusters", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingSSHKey != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingSSHKey)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.ExpectedResponse
			// since Cluster.Name is automatically generated by the system just rewrite it.
			if tc.RewriteClusterNameAndNamespaceName {
				actualCluster := &apiv1.Cluster{}
				err = json.Unmarshal(res.Body.Bytes(), actualCluster)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualCluster.Name, actualCluster.Status.NamespaceName)
			}

			compareWithResult(t, res, expectedResponse)
		})
	}
}

func TestClusterEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		clusterName     string
		responseCode    int
		cluster         *kubermaticv1.Cluster
		expectedCluster *kubermaticv1.Cluster
	}{
		{
			name:        "successful got cluster",
			clusterName: "foo",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			expectedCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusOK,
		},
		{
			name:        "unauthorized",
			clusterName: "foo",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": "not-current-user"},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusUnauthorized,
		},
		{
			name:        "not-found",
			clusterName: "not-existing-cluster",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "admintoken",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{},
			},
			responseCode: http.StatusNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster/"+test.clusterName, nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, []runtime.Object{test.cluster}, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)
			checkStatusCode(test.responseCode, res, t)

			if test.responseCode != http.StatusOK {
				return
			}

			gotCluster := &kubermaticv1.Cluster{}
			err = json.Unmarshal(res.Body.Bytes(), gotCluster)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotCluster, test.expectedCluster); diff != nil {
				t.Errorf("got different cluster than expected. Diff: %v", diff)
			}
		})
	}
}

func TestClustersEndpoint(t *testing.T) {
	t.Parallel()
	clusterList := []runtime.Object{
		&kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "cluster-user1-1",
				Labels: map[string]string{"user": testUsername},
			},
			Status: kubermaticv1.ClusterStatus{
				RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
			},
			Address: kubermaticv1.ClusterAddress{
				AdminToken: "admintoken",
				URL:        "https://foo.bar:8443",
			},
			Spec: kubermaticv1.ClusterSpec{},
		},
		&kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "cluster-user1-2",
				Labels: map[string]string{"user": testUsername},
			},
			Status: kubermaticv1.ClusterStatus{
				RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
			},
			Address: kubermaticv1.ClusterAddress{
				AdminToken: "admintoken",
				URL:        "https://foo.bar:8443",
			},
			Spec: kubermaticv1.ClusterSpec{},
		},
		&kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "cluster-user2-1",
				Labels: map[string]string{"user": "user2"},
			},
			Status: kubermaticv1.ClusterStatus{
				RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
			},
			Address: kubermaticv1.ClusterAddress{
				AdminToken: "admintoken",
				URL:        "https://foo.bar:8443",
			},
			Spec: kubermaticv1.ClusterSpec{},
		},
	}

	tests := []struct {
		name             string
		wantClusterNames []string
		admin            bool
		username         string
	}{
		{
			name:             "got user1 clusters",
			wantClusterNames: []string{"cluster-user1-1", "cluster-user1-2"},
			admin:            false,
			username:         testUsername,
		},
		{
			name:             "got user2 clusters",
			wantClusterNames: []string{"cluster-user2-1"},
			admin:            false,
			username:         "user2",
		},
		{
			name:             "got no cluster",
			wantClusterNames: []string{},
			admin:            false,
			username:         "does-not-exist",
		},
		{
			name:             "admin - got all cluster",
			wantClusterNames: []string{"cluster-user1-1", "cluster-user1-2", "cluster-user2-1"},
			admin:            true,
			username:         "foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster", nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(getUser(test.username, test.admin), []runtime.Object{}, clusterList, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			checkStatusCode(http.StatusOK, res, t)

			gotClusters := []kubermaticv1.Cluster{}
			err = json.Unmarshal(res.Body.Bytes(), &gotClusters)
			if err != nil {
				t.Fatal(err, res.Body.String())
			}

			gotClusterNames := []string{}
			for _, c := range gotClusters {
				gotClusterNames = append(gotClusterNames, c.Name)
			}

			if len(gotClusterNames) != len(test.wantClusterNames) {
				t.Errorf("got more/less clusters than expected. Got: %v Want: %v", gotClusterNames, test.wantClusterNames)
			}

			for _, w := range test.wantClusterNames {
				found := false
				for _, g := range gotClusterNames {
					if w == g {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("could not find cluster %s", w)
				}
			}
		})
	}
}

func TestClustersEndpointWithInvalidUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster", nil)
	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(getUser(strings.Repeat("A", 100), false), []runtime.Object{}, []runtime.Object{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	ep.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("got invalid status code. Expected 500, got: %d", res.Code)
	}

	s := res.Body.String()
	if !strings.Contains(s, "failed to create a valid cluster filter") {
		t.Fatalf("got unknown response error: %s", s)
	}
}

func TestUpdateClusterEndpoint(t *testing.T) {
	tests := []struct {
		name            string
		responseCode    int
		cluster         *kubermaticv1.Cluster
		expectedCluster *kubermaticv1.Cluster
		modifyCluster   func(*kubermaticv1.Cluster) *kubermaticv1.Cluster
	}{
		{
			name: "successful update admin token",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken:   "cccccc.cccccccccccccccc",
					KubeletToken: "cccccc.cccccccccccccccc",
					URL:          "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "foo",
						},
						DatacenterName: "us-central1",
					},
				},
			},
			responseCode: http.StatusOK,
			modifyCluster: func(c *kubermaticv1.Cluster) *kubermaticv1.Cluster {
				c.Address.AdminToken = "bbbbbb.bbbbbbbbbbbbbbbb"
				return c
			},
			expectedCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "foo",
						},
						DatacenterName: "us-central1",
					},
				},
			},
		},
		{
			name: "successful update cloud token",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken:   "cccccc.cccccccccccccccc",
					KubeletToken: "cccccc.cccccccccccccccc",
					URL:          "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "foo",
						},
						DatacenterName: "us-central1",
					},
				},
			},
			responseCode: http.StatusOK,
			modifyCluster: func(c *kubermaticv1.Cluster) *kubermaticv1.Cluster {
				c.Spec.Cloud.Fake.Token = "bar"
				return c
			},
			expectedCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "bar",
						},
						DatacenterName: "us-central1",
					},
				},
			},
		},
		{
			name: "invalid admin token",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken:   "cccccc.cccccccccccccccc",
					KubeletToken: "cccccc.cccccccccccccccc",
					URL:          "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "foo",
						},
						DatacenterName: "us-central1",
					},
				},
			},
			responseCode: http.StatusBadRequest,
			modifyCluster: func(c *kubermaticv1.Cluster) *kubermaticv1.Cluster {
				c.Address.AdminToken = "foo-bar"
				return c
			},
		},
		{
			name: "invalid address update",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken:   "cccccc.cccccccccccccccc",
					KubeletToken: "cccccc.cccccccccccccccc",
					URL:          "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: &kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "foo",
						},
						DatacenterName: "us-central1",
					},
				},
			},
			responseCode: http.StatusBadRequest,
			modifyCluster: func(c *kubermaticv1.Cluster) *kubermaticv1.Cluster {
				c.Address.URL = "https://example:8443"
				return c
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, []runtime.Object{test.cluster}, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			updatedCluster := test.cluster.DeepCopy()
			updatedCluster = test.modifyCluster(updatedCluster)
			body := &bytes.Buffer{}
			if err := json.NewEncoder(body).Encode(updatedCluster); err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("PUT", "/api/v3/dc/us-central1/cluster/"+test.cluster.Name, body)
			ep.ServeHTTP(res, req)
			checkStatusCode(test.responseCode, res, t)

			if test.responseCode != http.StatusOK {
				return
			}

			gotCluster := &kubermaticv1.Cluster{}
			err = json.Unmarshal(res.Body.Bytes(), gotCluster)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotCluster, test.expectedCluster); diff != nil {
				t.Errorf("got different cluster than expected. Diff: %v", diff)
			}
		})
	}
}
