package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/validation"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	clienttesting "k8s.io/client-go/testing"
)

func TestRemoveSensitiveDataFromCluster(t *testing.T) {
	t.Parallel()
	genClusterWithAdminToken := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = "hfzj6l.w7hgc65nq9z4fxvl"
		cluster.Address.ExternalName = "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io"
		cluster.Address.IP = "35.194.142.199"
		cluster.Address.URL = "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"
		return cluster
	}
	genClusterWithAWS := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = ""
		cluster.Spec.Cloud = kubermaticv1.CloudSpec{
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:         "secretKeyID",
				SecretAccessKey:     "secreatAccessKey",
				SecurityGroupID:     "secuirtyGroupID",
				AvailabilityZone:    "availablityZone",
				InstanceProfileName: "instanceProfileName",
				RoleName:            "roleName",
				RouteTableID:        "routeTableID",
				SubnetID:            "subnetID",
				VPCID:               "vpcID",
			},
		}

		return cluster
	}
	genClusterWithAzure := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = ""
		cluster.Spec.Cloud = kubermaticv1.CloudSpec{
			Azure: &kubermaticv1.AzureCloudSpec{
				ClientID:        "clientID",
				ClientSecret:    "clientSecret",
				TenantID:        "tenantID",
				AvailabilitySet: "availablitySet",
				ResourceGroup:   "resourceGroup",
				RouteTableName:  "routeTableName",
				SecurityGroup:   "securityGroup",
				SubnetName:      "subnetName",
				SubscriptionID:  "subsciprionID",
				VNetName:        "vnetname",
			},
		}
		return cluster
	}
	genClusterWithHetzner := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = ""
		cluster.Spec.Cloud = kubermaticv1.CloudSpec{
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token: "token",
			},
		}
		return cluster
	}
	genClusterWithDO := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = ""
		cluster.Spec.Cloud = kubermaticv1.CloudSpec{
			Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
				Token: "token",
			},
		}
		return cluster
	}
	genClusterWithVsphere := func() *kubermaticv1.Cluster {
		cluster := genDefaultCluster()
		cluster.Address.AdminToken = ""
		cluster.Spec.Cloud = kubermaticv1.CloudSpec{
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Password: "password",
				Username: "username",
				InfraManagementUser: kubermaticv1.VSphereCredentials{
					Username: "infraUsername",
					Password: "infraPassword",
				},
				VMNetName: "vmNetName",
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
		{
			Name:            "scenario 3: removes Azure cloud provider secrets",
			ExistingCluster: genClusterWithAzure(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithAzure()
				cluster.Spec.Cloud.Azure.ClientID = ""
				cluster.Spec.Cloud.Azure.ClientSecret = ""
				return cluster
			}(),
		},
		{
			Name:            "scenario 4: removes Openstack cloud provider secrets",
			ExistingCluster: genClusterWithOpenstack(genDefaultCluster()),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithOpenstack(genDefaultCluster())
				cluster.Address.AdminToken = ""
				cluster.Spec.Cloud.Openstack.Username = ""
				cluster.Spec.Cloud.Openstack.Password = ""
				return cluster
			}(),
		},
		{
			Name:            "scenario 5: removes Hetzner cloud provider secrets",
			ExistingCluster: genClusterWithHetzner(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithHetzner()
				cluster.Spec.Cloud.Hetzner.Token = ""
				return cluster
			}(),
		},
		{
			Name:            "scenario 6: removes Digitalocean cloud provider secrets",
			ExistingCluster: genClusterWithDO(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithDO()
				cluster.Spec.Cloud.Digitalocean.Token = ""
				return cluster
			}(),
		},
		{
			Name:            "scenario 7: removes Vsphere cloud provider secrets",
			ExistingCluster: genClusterWithVsphere(),
			ExpectedCluster: func() *kubermaticv1.Cluster {
				cluster := genClusterWithVsphere()
				cluster.Spec.Cloud.VSphere.Username = ""
				cluster.Spec.Cloud.VSphere.Password = ""
				cluster.Spec.Cloud.VSphere.InfraManagementUser.Password = ""
				cluster.Spec.Cloud.VSphere.InfraManagementUser.Username = ""
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

func TestDeleteClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcase := struct {
		Name                          string
		Body                          string
		ExpectedResponse              string
		HTTPStatus                    int
		ProjectToSync                 string
		ClusterToSync                 string
		ExistingProject               *kubermaticv1.Project
		ExistingKubermaticUser        *kubermaticv1.User
		ExistingAPIUser               *apiv1.User
		ExistingCluster               *kubermaticv1.Cluster
		ExistingSSHKeys               []*kubermaticv1.UserSSHKey
		ExpectedSSHKeys               []*kubermaticv1.UserSSHKey
		ExpectedListClusterKeysStatus int
	}{
		Name:             "scenario 1: tests deletion of a cluster and its dependant resources",
		Body:             ``,
		ExpectedResponse: `{}`,
		HTTPStatus:       http.StatusOK,
		ExistingProject:  genDefaultProject(),
		ProjectToSync:    genDefaultProject().Name,
		ExistingKubermaticUser: &kubermaticv1.User{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: kubermaticv1.UserSpec{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
				Projects: []kubermaticv1.ProjectGroup{
					{
						Group: "owners-" + genDefaultProject().Name,
						Name:  genDefaultProject().Name,
					},
				},
			},
		},
		ExistingAPIUser: &apiv1.User{
			Name:  testUserName,
			ID:    testUserID,
			Email: testUserEmail,
		},
		ExistingSSHKeys: []*kubermaticv1.UserSSHKey{
			&kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
				Spec: kubermaticv1.SSHKeySpec{
					Clusters: []string{"clusterAbcID"},
				},
			},
			&kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-abc-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
				Spec: kubermaticv1.SSHKeySpec{
					Clusters: []string{"clusterAbcID"},
				},
			},
		},

		ExpectedSSHKeys: []*kubermaticv1.UserSSHKey{
			&kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
				Spec: kubermaticv1.SSHKeySpec{
					Clusters: []string{},
				},
			},
			&kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-abc-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
				Spec: kubermaticv1.SSHKeySpec{
					Clusters: []string{},
				},
			},
		},
		ExistingCluster:               genCluster("clusterAbcID", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
		ClusterToSync:                 "clusterAbcID",
		ExpectedListClusterKeysStatus: http.StatusNotFound,
	}

	// validate if deletion was successful
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", testcase.ProjectToSync, testcase.ClusterToSync), strings.NewReader(testcase.Body))
	res := httptest.NewRecorder()
	kubermaticObj := []runtime.Object{}
	kubermaticObj = append(kubermaticObj, testcase.ExistingProject)
	kubermaticObj = append(kubermaticObj, testcase.ExistingCluster)
	kubermaticObj = append(kubermaticObj, testcase.ExistingKubermaticUser)
	for _, existingKey := range testcase.ExistingSSHKeys {
		kubermaticObj = append(kubermaticObj, existingKey)
	}

	ep, clientsSets, err := createTestEndpointAndGetClients(*testcase.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	kubermaticClient := clientsSets.fakeKubermaticClient

	ep.ServeHTTP(res, req)

	if res.Code != testcase.HTTPStatus {
		t.Fatalf("Expected HTTP status code %d, got %d: %s", testcase.HTTPStatus, res.Code, res.Body.String())
	}
	compareWithResult(t, res, testcase.ExpectedResponse)

	validatedActions := 0
	for _, action := range kubermaticClient.Actions() {
		if action.Matches("update", "usersshkeies") {
			updateAction, ok := action.(clienttesting.CreateAction)
			if !ok {
				t.Fatalf("unexpected action %#v", action)
			}
			for _, expectedSSHKey := range testcase.ExpectedSSHKeys {
				sshKeyFromAction := updateAction.GetObject().(*kubermaticv1.UserSSHKey)
				if sshKeyFromAction.Name == expectedSSHKey.Name {
					if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.UserSSHKey), expectedSSHKey) {
						t.Fatalf("%v", diff.ObjectDiff(expectedSSHKey, updateAction.GetObject().(*kubermaticv1.UserSSHKey)))
					}
				}
			}
			validatedActions = validatedActions + 1
		}
	}
	if validatedActions != len(testcase.ExpectedSSHKeys) {
		t.Fatalf("not all update actions were validated, expected to validate %d but validated only %d", len(testcase.ExpectedSSHKeys), validatedActions)
	}

	// validate if the cluster was deleted
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/sshkeys", testcase.ProjectToSync), strings.NewReader(testcase.Body))
	res = httptest.NewRecorder()
	ep.ServeHTTP(res, req)
	if res.Code != testcase.ExpectedListClusterKeysStatus {
		t.Fatalf("Expected HTTP status code %d, got %d: %s", testcase.ExpectedListClusterKeysStatus, res.Code, res.Body.String())
	}
}

func TestDetachSSHKeyFromClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                            string
		Body                            string
		KeyToDelete                     string
		ProjectToSync                   string
		ClusterToSync                   string
		ExpectedDeleteResponse          string
		ExpectedDeleteHTTPStatus        int
		ExistingProject                 *kubermaticv1.Project
		ExistingKubermaticUser          *kubermaticv1.User
		ExistingAPIUser                 *apiv1.User
		ExistingCluster                 *kubermaticv1.Cluster
		ExistingSSHKeys                 []*kubermaticv1.UserSSHKey
		ExpectedResponseOnGetAfterDelte string
		ExpectedGetHTTPStatus           int
	}{
		// scenario 1
		{
			Name:                            "scenario 1: detaches one key from the cluster",
			Body:                            ``,
			KeyToDelete:                     "key-c08aa5c7abf34504f18552846485267d-yafn",
			ExpectedDeleteResponse:          `{}`,
			ExpectedDeleteHTTPStatus:        http.StatusOK,
			ExpectedGetHTTPStatus:           http.StatusOK,
			ExpectedResponseOnGetAfterDelte: `[{"id":"key-abc-yafn","name":"key-display-name","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"","publicKey":""}}]`,
			ExistingProject:                 genDefaultProject(),
			ProjectToSync:                   genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingSSHKeys: []*kubermaticv1.UserSSHKey{
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       genDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{"clusterAbcID"},
					},
				},
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       genDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "key-display-name",
						Clusters: []string{"clusterAbcID"},
					},
				},
			},
			ExistingCluster: genCluster("clusterAbcID", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			ClusterToSync:   "clusterAbcID",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var ep http.Handler
			{
				var err error
				req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys/%s", tc.ProjectToSync, tc.ClusterToSync, tc.KeyToDelete), strings.NewReader(tc.Body))
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
				for _, existingKey := range tc.ExistingSSHKeys {
					kubermaticObj = append(kubermaticObj, existingKey)
				}
				ep, err = createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
				if err != nil {
					t.Fatalf("failed to create test endpoint due to %v", err)
				}

				ep.ServeHTTP(res, req)

				if res.Code != tc.ExpectedDeleteHTTPStatus {
					t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedDeleteHTTPStatus, res.Code, res.Body.String())
				}
				compareWithResult(t, res, tc.ExpectedDeleteResponse)
			}

			// GET request list the keys from the cache, thus we wait 1 s before firing the request . . . I know :)
			time.Sleep(time.Second)
			{
				req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(tc.Body))
				res := httptest.NewRecorder()

				ep.ServeHTTP(res, req)

				if res.Code != tc.ExpectedGetHTTPStatus {
					t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedGetHTTPStatus, res.Code, res.Body.String())
				}
				compareWithResult(t, res, tc.ExpectedResponseOnGetAfterDelte)
			}
		})
	}
}

func TestListSSHKeysAssignedToClusterEndpoint(t *testing.T) {
	t.Parallel()
	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	creationTime, err := time.Parse(longForm, "Feb 3, 2013 at 7:54pm (PST)")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                   string
		Body                   string
		ProjectToSync          string
		ClusterToSync          string
		ExpectedKeys           []apiv1.NewSSHKey
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name: "scenario 1: gets a list of ssh keys assigned to cluster",
			Body: ``,
			ExpectedKeys: []apiv1.NewSSHKey{
				apiv1.NewSSHKey{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:                "key-c08aa5c7abf34504f18552846485267d-yafn",
						Name:              "yafn",
						CreationTimestamp: time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				apiv1.NewSSHKey{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:                "key-abc-yafn",
						Name:              "abcd",
						CreationTimestamp: time.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingProject: genDefaultProject(),
			ProjectToSync:   genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingSSHKeys: []*kubermaticv1.UserSSHKey{
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       genDefaultProject().Name,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "yafn",
						Clusters: []string{genDefaultCluster().Name},
					},
				},
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       genDefaultProject().Name,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime.Add(time.Minute)),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "abcd",
						Clusters: []string{genDefaultCluster().Name},
					},
				},
			},
			ExistingCluster: genDefaultCluster(),
			ClusterToSync:   genDefaultCluster().Name,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(tc.Body))
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
			for _, existingKey := range tc.ExistingSSHKeys {
				kubermaticObj = append(kubermaticObj, existingKey)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualKeys := newSSHKeyV1SliceWrapper{}
			actualKeys.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedKeys := newSSHKeyV1SliceWrapper(tc.ExpectedKeys)
			wrappedExpectedKeys.Sort()

			actualKeys.EqualOrDie(wrappedExpectedKeys, t)
		})
	}
}

func TestAssignSSHKeyToClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		SSHKeyID               string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ClusterToSync          string
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKey         *kubermaticv1.UserSSHKey
		ExpectedSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:             "scenario 1: an ssh key that belongs to the given project is assigned to the cluster",
			SSHKeyID:         "key-c08aa5c7abf34504f18552846485267d-yafn",
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExpectedSSHKeys: []*kubermaticv1.UserSSHKey{
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       genDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{genDefaultCluster().Name},
					},
				},
			},
			ExistingSSHKey: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
			},
			ExistingCluster: genDefaultCluster(),
			ClusterToSync:   genDefaultCluster().Name,
		},
		// scenario 2
		{
			Name:             "scenario 2: an ssh key that does not belong to the given project cannot be assigned to the cluster",
			SSHKeyID:         "key-c08aa5c7abf34504f18552846485267d-yafn",
			ExpectedResponse: `{"error":{"code":500,"message":"the given ssh key key-c08aa5c7abf34504f18552846485267d-yafn does not belong to the given project my-first-project (my-first-projectInternalName)"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExpectedSSHKeys: []*kubermaticv1.UserSSHKey{},
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
			ExistingCluster: genDefaultCluster(),
			ClusterToSync:   genDefaultCluster().Name,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys/%s", tc.ProjectToSync, tc.ClusterToSync, tc.SSHKeyID), nil)
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
			ep, clientsSets, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			compareWithResult(t, res, tc.ExpectedResponse)

			kubermaticClient := clientsSets.fakeKubermaticClient
			validatedActions := 0
			if tc.HTTPStatus == http.StatusCreated {
				for _, action := range kubermaticClient.Actions() {
					if action.Matches("update", "usersshkeies") {
						updateAction, ok := action.(clienttesting.CreateAction)
						if !ok {
							t.Fatalf("unexpected action %#v", action)
						}
						for _, expectedSSHKey := range tc.ExpectedSSHKeys {
							sshKeyFromAction := updateAction.GetObject().(*kubermaticv1.UserSSHKey)
							if sshKeyFromAction.Name == expectedSSHKey.Name {
								validatedActions = validatedActions + 1
								if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.UserSSHKey), expectedSSHKey) {
									t.Fatalf("%v", diff.ObjectDiff(expectedSSHKey, updateAction.GetObject().(*kubermaticv1.UserSSHKey)))
								}
							}
						}
					}
				}
				if validatedActions != len(tc.ExpectedSSHKeys) {
					t.Fatalf("not all update actions were validated, expected to validate %d but validated only %d", len(tc.ExpectedSSHKeys), validatedActions)
				}
			}
		})
	}
}

func TestCreateClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingSSHKey         *kubermaticv1.UserSSHKey
		RewriteClusterID       bool
	}{
		// scenario 1
		{
			Name:             "scenario 1: a cluster with invalid spec is rejected",
			Body:             `{"name":"keen-snyder","spec":{"cloud":{"digitalocean":{"token":"dummy_token"},"dc":"do-fra1"}, "version":""}}`,
			ExpectedResponse: `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec \"Version\" is required but was not specified"}}`,
			HTTPStatus:       http.StatusBadRequest,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  testUserName,
					ID:    testUserID,
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
		},
		// scenario 2
		{
			Name:             "scenario 2: cluster is created when valid spec and ssh key are passed",
			Body:             `{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"do-fra1"}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"cloud":{"dc":"do-fra1","fake":{"token":"dummy_token"}},"version":"1.9.7"},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  testUserName,
					ID:    testUserID,
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				Name:  testUserName,
				ID:    testUserID,
				Email: testUserEmail,
			},
			ExistingSSHKey: &kubermaticv1.UserSSHKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       genDefaultProject().Name,
						},
					},
				},
			},
		},
		// scenario 3
		{
			Name:             "scenario 3: unable to create a cluster when the user doesn't belong to the project",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"digitalocean":{"token":"dummy_token"},"dc":"do-fra1"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: The user \"john@acme.com\" doesn't belong to the given project = my-first-projectInternalName"}}`,
			HTTPStatus:       http.StatusForbidden,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  testUserName,
					ID:    testUserID,
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-secretProject",
							Name:  "secretProject",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
		},
		// scenario 4
		{
			Name:             "scenario 4: unable to create a cluster when project is not ready",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"digitalocean":{"token":"dummy_token"},"dc":"do-fra1"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":503,"message":"Project is not initialized yet"}}`,
			HTTPStatus:       http.StatusServiceUnavailable,
			ExistingProject: func() *kubermaticv1.Project {
				project := genDefaultProject()
				project.Status.Phase = kubermaticv1.ProjectInactive
				return project
			}(),
			ProjectToSync: genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  testUserName,
					ID:    testUserID,
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: fmt.Sprintf("owners-%s", genDefaultProject().Name),
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserID,
				Name:  testUserName,
				Email: testUserEmail,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters", tc.ProjectToSync), strings.NewReader(tc.Body))
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
			if tc.RewriteClusterID {
				actualCluster := &apiv1.NewCluster{}
				err = json.Unmarshal(res.Body.Bytes(), actualCluster)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualCluster.ID)
			}

			compareWithResult(t, res, expectedResponse)
		})
	}
}

func TestGetClusterHealth(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingClusters       []*kubermaticv1.Cluster
	}{
		// scenario 1
		{
			Name:             "scenario 1: get existing cluster health status",
			Body:             ``,
			ExpectedResponse: `{"apiserver":true,"scheduler":false,"controller":true,"machineController":false,"etcd":true}`,
			HTTPStatus:       http.StatusOK,
			ClusterToGet:     "keen-snyder",
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingClusters: []*kubermaticv1.Cluster{
				func() *kubermaticv1.Cluster {
					cluster := genCluster("keen-snyder", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Status.Health = kubermaticv1.ClusterHealth{
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
							Apiserver:         true,
							Scheduler:         false,
							Controller:        true,
							MachineController: false,
							Etcd:              true,
						},
					}
					return cluster
				}(),
				genCluster("clusterDefID", "clusterDef", genDefaultProject().Name, time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC)),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/health", tc.ProjectToSync, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			for _, existingCluster := range tc.ExistingClusters {
				kubermaticObj = append(kubermaticObj, existingCluster)
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

			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestUpdateCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToUpdate        string
		ProjectToSync          string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingClusters       []*kubermaticv1.Cluster
	}{
		// scenario 1
		{
			Name:             "scenario 1: update the cluster version",
			Body:             `{"name":"keen-snyder","spec":{"version":"0.0.1","cloud":{"fake":{"token":"dummy_token"},"dc":"do-fra1"}}}`,
			ExpectedResponse: `{"id":"keen-snyder","name":"clusterAbc","creationTimestamp":"2013-02-03T19:54:00Z","spec":{"cloud":{"dc":"do-fra1","fake":{"token":"dummy_token"}},"version":"0.0.1"},"status":{"version":"0.0.1","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToUpdate:  "keen-snyder",
			HTTPStatus:       http.StatusOK,
			ExistingProject:  genDefaultProject(),
			ProjectToSync:    genDefaultProject().Name,
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingClusters: []*kubermaticv1.Cluster{
				genCluster("keen-snyder", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				genDefaultCluster(),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", tc.ProjectToSync, tc.ClusterToUpdate), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			for _, existingCluster := range tc.ExistingClusters {
				kubermaticObj = append(kubermaticObj, existingCluster)
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

			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestGetCluster(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingClusters       []*kubermaticv1.Cluster
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets cluster with the given name that belongs to the given project",
			Body:             ``,
			ExpectedResponse: `{"id":"clusterID","name":"clusterName","creationTimestamp":"2013-02-03T19:54:00Z","spec":{"cloud":{"dc":"FakeDatacenter","fake":{"token":"SecretToken"}},"version":"9.9.9"},"status":{"version":"9.9.9","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToGet:     genDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingProject:  genDefaultProject(),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingClusters: []*kubermaticv1.Cluster{
				genDefaultCluster(),
				genCluster("clusterAbcID", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			},
		},
		// scenario 2
		{
			Name:             "scenario 2: gets cluster for Openstack and no sensitive data (credentials) are returned",
			Body:             ``,
			ExpectedResponse: `{"id":"clusterID","name":"clusterName","creationTimestamp":"2013-02-03T19:54:00Z","spec":{"cloud":{"dc":"OpenstackDatacenter","openstack":{"username":"","password":"","tenant":"tenant","domain":"domain","network":"network","securityGroups":"securityGroups","floatingIpPool":"floatingIPPool","routerID":"routerID","subnetID":"subnetID"}},"version":"9.9.9"},"status":{"version":"9.9.9","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToGet:     genDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingProject:  genDefaultProject(),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + genDefaultProject().Name,
							Name:  genDefaultProject().Name,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingClusters: []*kubermaticv1.Cluster{
				genClusterWithOpenstack(genDefaultCluster()),
				genCluster("clusterAbcID", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", testingProjectName, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			for _, existingCluster := range tc.ExistingClusters {
				kubermaticObj = append(kubermaticObj, existingCluster)
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

			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListClusters(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedClusters       []apiv1.NewCluster
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingClusters       []*kubermaticv1.Cluster
	}{
		// scenario 1
		{
			Name: "scenario 1: list clusters that belong to the given project",
			Body: ``,
			ExpectedClusters: []apiv1.NewCluster{
				apiv1.NewCluster{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:                "clusterAbcID",
						Name:              "clusterAbc",
						CreationTimestamp: time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.NewClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake: &kubermaticv1.FakeCloudSpec{
								Token: "SecretToken",
							},
						},
						Version: "9.9.9",
					},
					Status: apiv1.NewClusterStatus{
						Version: "9.9.9",
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
				},
				apiv1.NewCluster{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:                "clusterDefID",
						Name:              "clusterDef",
						CreationTimestamp: time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.NewClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake: &kubermaticv1.FakeCloudSpec{
								Token: "SecretToken",
							},
						},
						Version: "9.9.9",
					},
					Status: apiv1.NewClusterStatus{
						Version: "9.9.9",
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
				},
				apiv1.NewCluster{
					NewObjectMeta: apiv1.NewObjectMeta{
						ID:                "clusterOpenstackID",
						Name:              "clusterOpenstack",
						CreationTimestamp: time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.NewClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "OpenstackDatacenter",
							Openstack: func() *kubermaticv1.OpenstackCloudSpec {
								cluster := genClusterWithOpenstack(genDefaultCluster())
								cluster.Spec.Cloud.Openstack.Password = ""
								cluster.Spec.Cloud.Openstack.Username = ""
								return cluster.Spec.Cloud.Openstack
							}(),
						},
						Version: "9.9.9",
					},
					Status: apiv1.NewClusterStatus{
						Version: "9.9.9",
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
				},
			},
			HTTPStatus:      http.StatusOK,
			ExistingProject: genDefaultProject(),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingClusters: []*kubermaticv1.Cluster{
				genCluster("clusterAbcID", "clusterAbc", genDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				genCluster("clusterDefID", "clusterDef", genDefaultProject().Name, time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC)),
				genClusterWithOpenstack(genCluster("clusterOpenstackID", "clusterOpenstack", genDefaultProject().Name, time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC))),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			for _, existingCluster := range tc.ExistingClusters {
				kubermaticObj = append(kubermaticObj, existingCluster)
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

			actualClusters := newClusterV1SliceWrapper{}
			actualClusters.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedClusters := newClusterV1SliceWrapper(tc.ExpectedClusters)
			wrappedExpectedClusters.Sort()

			actualClusters.EqualOrDie(wrappedExpectedClusters, t)
		})
	}
}

func TestLegacyClusterEndpoint(t *testing.T) {
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
					Labels: map[string]string{"user": testUserID},
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
					Labels: map[string]string{"user": testUserID},
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
					Labels: map[string]string{"user": testUserID},
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
			apiUser := getUser(testUserEmail, testUserID, testUserName, false)

			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster/"+test.clusterName, nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{test.cluster, apiUserToKubermaticUser(apiUser)}, nil, nil)
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

func TestLegacyClustersEndpoint(t *testing.T) {
	t.Parallel()
	clusterList := []runtime.Object{
		&kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "cluster-user1-1",
				Labels: map[string]string{"user": testUserID},
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
				Labels: map[string]string{"user": testUserID},
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
				Labels: map[string]string{"user": "666"},
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
		userid           string
		useremail        string
	}{
		{
			name:             "got user1 clusters",
			wantClusterNames: []string{"cluster-user1-1", "cluster-user1-2"},
			admin:            false,
			username:         testUserName,
			userid:           testUserID,
			useremail:        testUserEmail,
		},
		{
			name:             "got user2 clusters",
			wantClusterNames: []string{"cluster-user2-1"},
			admin:            false,
			username:         "user2",
			userid:           "666",
			useremail:        "a@abcd.com",
		},
		{
			name:             "got no cluster",
			wantClusterNames: []string{},
			admin:            false,
			username:         "does-not-exist",
			userid:           "007",
			useremail:        "bond@bond.com",
		},
		{
			name:             "admin - got all cluster",
			wantClusterNames: []string{"cluster-user1-1", "cluster-user1-2", "cluster-user2-1"},
			admin:            true,
			username:         "foo",
			userid:           "bar",
			useremail:        "foo@bar.com",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apiUser := getUser(test.useremail, test.userid, test.username, test.admin)
			kubermaticObjs := []runtime.Object{}
			kubermaticObjs = append(kubermaticObjs, clusterList...)
			kubermaticObjs = append(kubermaticObjs, apiUserToKubermaticUser(apiUser))

			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster", nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(apiUser, []runtime.Object{}, kubermaticObjs, nil, nil)
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

func TestLegacyClustersEndpointWithInvalidUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster", nil)
	res := httptest.NewRecorder()
	ep, err := createTestEndpoint(getUser("foo", strings.Repeat("A", 100), "some-email@loodse.com", false), []runtime.Object{}, []runtime.Object{}, nil, nil)
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

func TestLegacyUpdateClusterEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		responseCode    int
		cluster         *kubermaticv1.Cluster
		expectedCluster *kubermaticv1.Cluster
		modifyCluster   func(*kubermaticv1.Cluster) *kubermaticv1.Cluster
	}{
		{
			name: "successful update admin token (deprecated)",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "cccccc.cccccccccccccccc",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
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
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
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
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "cccccc.cccccccccccccccc",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
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
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					URL: "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "bar",
						},
						DatacenterName: "us-central1",
					},
				},
			},
		},
		{
			name: "invalid admin token (deprecated)",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "cccccc.cccccccccccccccc",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
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
					Labels: map[string]string{"user": testUserID},
				},
				Status: kubermaticv1.ClusterStatus{
					RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
				},
				Address: kubermaticv1.ClusterAddress{
					AdminToken: "cccccc.cccccccccccccccc",
					URL:        "https://foo.bar:8443",
				},
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
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
			apiUser := getUser(testUserEmail, testUserID, testUserName, false)

			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(apiUser, []runtime.Object{}, []runtime.Object{test.cluster, apiUserToKubermaticUser(apiUser)}, nil, nil)
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

func TestGetClusterAdminTokenEndpoint(t *testing.T) {
	t.Parallel()
	tester := apiv1.User{
		ID:    testUserName,
		Email: testUserEmail,
	}

	user := &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.UserSpec{
			Name:  "John",
			Email: testUserEmail,
			Projects: []kubermaticv1.ProjectGroup{
				{
					Group: "owners-" + testingProjectName,
					Name:  testingProjectName,
				},
			},
		},
	}

	project := genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp())

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "foo",
			Labels: map[string]string{"user": testUserName},
		},
		Status: kubermaticv1.ClusterStatus{
			RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken: "cccccc.cccccccccccccccc",
			URL:        "https://foo.bar:8443",
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				Fake: &kubermaticv1.FakeCloudSpec{
					Token: "foo",
				},
				DatacenterName: "us-central1",
			},
		},
	}

	expectedResponse := fmt.Sprintf(`{"token":"%s"}`, cluster.Address.AdminToken)

	// setup world view
	ep, err := createTestEndpoint(tester, []runtime.Object{}, []runtime.Object{user, project, cluster}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	// perform test
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/token", testingProjectName, cluster.Name), nil)
	ep.ServeHTTP(res, req)

	// check assertions
	checkStatusCode(http.StatusOK, res, t)
	compareWithResult(t, res, expectedResponse)
}

func TestRevokeClusterAdminTokenEndpoint(t *testing.T) {
	t.Parallel()
	tester := apiv1.User{
		ID:    testUserName,
		Email: testUserEmail,
	}

	user := &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: kubermaticv1.UserSpec{
			Name:  "John",
			Email: testUserEmail,
			Projects: []kubermaticv1.ProjectGroup{
				{
					Group: "owners-" + testingProjectName,
					Name:  testingProjectName,
				},
			},
		},
	}

	project := genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp())

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "foo",
			Labels: map[string]string{"user": testUserName},
		},
		Status: kubermaticv1.ClusterStatus{
			RootCA: kubermaticv1.KeyCert{Cert: []byte("foo")},
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken: "cccccc.cccccccccccccccc",
			URL:        "https://foo.bar:8443",
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				Fake: &kubermaticv1.FakeCloudSpec{
					Token: "foo",
				},
				DatacenterName: "us-central1",
			},
		},
	}

	// setup world view
	ep, clientsSets, err := createTestEndpointAndGetClients(tester, nil, []runtime.Object{}, []runtime.Object{}, []runtime.Object{user, project, cluster}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	// perform test
	res := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/token", testingProjectName, cluster.Name), nil)
	ep.ServeHTTP(res, req)

	// check assertions
	checkStatusCode(http.StatusOK, res, t)

	response := &apiv1.ClusterAdminToken{}
	err = json.Unmarshal(res.Body.Bytes(), response)
	if err != nil {
		t.Fatal(err)
	}

	if len(response.Token) == 0 || response.Token == cluster.Address.AdminToken {
		t.Errorf("revocation response does not contain updated admin token, but '%s'", response.Token)
	}

	if err := validation.ValidateKubernetesToken(response.Token); err != nil {
		t.Errorf("generated token '%s' is malformed: %v", response.Token, err)
	}

	// check if the cluster was really updated
	updatedCluster, err := clientsSets.fakeKubermaticClient.KubermaticV1().Clusters().Get(cluster.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if updatedCluster.Address.AdminToken == cluster.Address.AdminToken {
		t.Error("updated admin token in cluster resource was not persisted")
	}
}

func genCluster(id string, name string, projectID string, creationTime time.Time) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: map[string]string{"project-id": projectID},
			CreationTimestamp: func() metav1.Time {
				return metav1.NewTime(creationTime)
			}(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "kubermatic.k8s.io/v1",
					Kind:       "Project",
					UID:        "",
					Name:       projectID,
				},
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "FakeDatacenter",
				Fake:           &kubermaticv1.FakeCloudSpec{Token: "SecretToken"},
			},
			Version:           "9.9.9",
			HumanReadableName: name,
		},
		Address: kubermaticv1.ClusterAddress{
			AdminToken:   "drphc2.g4kq82pnlfqjqt65",
			ExternalName: "w225mx4z66.asia-east1-a-1.cloud.kubermatic.io",
			IP:           "35.194.142.199",
			URL:          "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
		},
		Status: kubermaticv1.ClusterStatus{
			Health: kubermaticv1.ClusterHealth{
				ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
					Apiserver:         true,
					Scheduler:         true,
					Controller:        true,
					MachineController: true,
					Etcd:              true,
				},
			},
		},
	}
}

func genDefaultCluster() *kubermaticv1.Cluster {
	return genCluster("clusterID", "clusterName", "projectID", time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
}

func genClusterWithOpenstack(cluster *kubermaticv1.Cluster) *kubermaticv1.Cluster {
	cluster.Spec.Cloud = kubermaticv1.CloudSpec{
		DatacenterName: "OpenstackDatacenter",
		Openstack: &kubermaticv1.OpenstackCloudSpec{
			Username:       "username",
			Password:       "password",
			SubnetID:       "subnetID",
			Domain:         "domain",
			FloatingIPPool: "floatingIPPool",
			Network:        "network",
			RouterID:       "routerID",
			SecurityGroups: "securityGroups",
			Tenant:         "tenant",
		},
	}
	return cluster
}
