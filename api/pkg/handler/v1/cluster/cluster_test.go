package cluster_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/handler/test/hack"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

const fakeDC = "fake-dc"

func TestDeleteClusterEndpointWithFinalizers(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExistingKubermaticObjs []runtime.Object
		HeaderParams           map[string]string
		ExpectedFinalizers     []string
	}{
		{
			Name: "scenario 1: tests deletion of a cluster with finalizers",
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Finalizers = []string{"kubermatic.io/delete-nodes"}
				}),
			),
			HeaderParams:       map[string]string{"DeleteVolumes": "true", "DeleteLoadBalancers": "true"},
			ExpectedFinalizers: []string{"kubermatic.io/cleanup-in-cluster-pv", "kubermatic.io/cleanup-in-cluster-lb", "kubermatic.io/delete-nodes"},
		},
		{
			Name: "scenario 2: tests deletion of a cluster with only volume finalizer",
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC), func(cluster *kubermaticv1.Cluster) {
					cluster.Finalizers = []string{"kubermatic.io/delete-nodes"}
				}),
			),
			HeaderParams:       map[string]string{"DeleteVolumes": "true", "DeleteLoadBalancers": "false"},
			ExpectedFinalizers: []string{"kubermatic.io/cleanup-in-cluster-pv", "kubermatic.io/delete-nodes"},
		},
		{
			Name: "scenario 3: tests deletion of a cluster without finalizers",
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			HeaderParams:       map[string]string{},
			ExpectedFinalizers: []string{},
		},
		{
			Name: "PV and LB finalizers do not get attached when cluster has no node delete finalizer",
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			HeaderParams:       map[string]string{"DeleteVolumes": "true", "DeleteLoadBalancers": "true"},
			ExpectedFinalizers: []string{},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubermaticObj []runtime.Object
			// validate if deletion was successful
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", test.GenDefaultProject().Name, "clusterAbcID"), strings.NewReader(""))

			for k, v := range tc.HeaderParams {
				req.Header.Add(k, v)
			}

			res := httptest.NewRecorder()
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*test.GenDefaultAPIUser(), nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			kubermaticClient := clientsSets.FakeKubermaticClient

			ep.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", http.StatusOK, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, "{}")

			validatedActions := 0
			for _, action := range kubermaticClient.Actions() {
				if action.Matches("update", "clusters") {
					updateaction, ok := action.(clienttesting.UpdateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}

					cluster := updateaction.GetObject().(*kubermaticv1.Cluster)
					finalizers := cluster.GetFinalizers()
					if len(finalizers) != len(tc.ExpectedFinalizers) {
						t.Fatalf("expected %d finalizers, got %d", len(tc.ExpectedFinalizers), len(finalizers))
					}

					sort.Strings(finalizers)
					sort.Strings(tc.ExpectedFinalizers)
					if !equality.Semantic.DeepEqual(finalizers, tc.ExpectedFinalizers) {
						t.Fatalf("finalizer list %v is not the same as expected %v", finalizers, tc.ExpectedFinalizers)
					}

					validatedActions++
				}
			}
			if validatedActions != 1 {
				t.Fatalf("expected 1 update, got %d", validatedActions)
			}
		})
	}
}

func TestDeleteClusterEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                          string
		Body                          string
		ExpectedResponse              string
		HTTPStatus                    int
		ProjectToSync                 string
		ClusterToSync                 string
		ExistingKubermaticObjs        []runtime.Object
		ExistingAPIUser               *apiv1.User
		ExpectedSSHKeys               []*kubermaticv1.UserSSHKey
		ExpectedListClusterKeysStatus int
	}{
		{
			Name:             "scenario 1: tests deletion of a cluster and its dependant resources",
			Body:             ``,
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				// add ssh keys
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
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
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{"clusterAbcID"},
					},
				},
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExpectedSSHKeys: []*kubermaticv1.UserSSHKey{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{},
					},
				},
			},
			ExpectedListClusterKeysStatus: http.StatusNotFound,
		},
		{
			Name:             "scenario 2: the admin John can delete Bob's cluster",
			Body:             ``,
			ExpectedResponse: `{}`,
			HTTPStatus:       http.StatusOK,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				// add ssh keys
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
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
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{"clusterAbcID"},
					},
				},
			),
			ClusterToSync:   "clusterAbcID",
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
			ExpectedSSHKeys: []*kubermaticv1.UserSSHKey{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{},
					},
				},
			},
			ExpectedListClusterKeysStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {

			// validate if deletion was successful
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			kubermaticClient := clientsSets.FakeKubermaticClient

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			test.CompareWithResult(t, res, tc.ExpectedResponse)

			validatedActions := 0
			for _, action := range kubermaticClient.Actions() {
				if action.Matches("update", "usersshkeies") {
					updateAction, ok := action.(clienttesting.CreateAction)
					if !ok {
						t.Fatalf("unexpected action %#v", action)
					}
					for _, expectedSSHKey := range tc.ExpectedSSHKeys {
						sshKeyFromAction := updateAction.GetObject().(*kubermaticv1.UserSSHKey)
						if sshKeyFromAction.Name == expectedSSHKey.Name {
							if !equality.Semantic.DeepEqual(updateAction.GetObject().(*kubermaticv1.UserSSHKey), expectedSSHKey) {
								t.Fatalf("%v", diff.ObjectDiff(expectedSSHKey, updateAction.GetObject().(*kubermaticv1.UserSSHKey)))
							}
						}
					}
					validatedActions++
				}
			}
			if validatedActions != len(tc.ExpectedSSHKeys) {
				t.Fatalf("not all update actions were validated, expected to validate %d but validated only %d", len(tc.ExpectedSSHKeys), validatedActions)
			}

			// validate if the cluster was deleted
			req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/abcd/sshkeys", tc.ProjectToSync), strings.NewReader(tc.Body))
			res = httptest.NewRecorder()
			ep.ServeHTTP(res, req)
			if res.Code != tc.ExpectedListClusterKeysStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedListClusterKeysStatus, res.Code, res.Body.String())
			}
		})
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
		ExistingAPIUser                 *apiv1.User
		ExistingSSHKeys                 []*kubermaticv1.UserSSHKey
		ExistingKubermaticObjs          []runtime.Object
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
			ProjectToSync:                   test.GenDefaultProject().Name,
			ExistingAPIUser:                 test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				// add ssh keys
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
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
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "key-display-name",
						Clusters: []string{"clusterAbcID"},
					},
				},
			),
			ClusterToSync: "clusterAbcID",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var ep http.Handler
			{
				var err error
				req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys/%s", tc.ProjectToSync, tc.ClusterToSync, tc.KeyToDelete), strings.NewReader(tc.Body))
				res := httptest.NewRecorder()
				var kubermaticObj []runtime.Object
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
				ep, err = test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
				if err != nil {
					t.Fatalf("failed to create test endpoint due to %v", err)
				}

				ep.ServeHTTP(res, req)

				if res.Code != tc.ExpectedDeleteHTTPStatus {
					t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedDeleteHTTPStatus, res.Code, res.Body.String())
				}
				test.CompareWithResult(t, res, tc.ExpectedDeleteResponse)
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
				test.CompareWithResult(t, res, tc.ExpectedResponseOnGetAfterDelte)
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
		ExpectedKeys           []apiv1.SSHKey
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: gets a list of ssh keys assigned to cluster",
			Body: ``,
			ExpectedKeys: []apiv1.SSHKey{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-c08aa5c7abf34504f18552846485267d-yafn",
						Name:              "yafn",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-abc-yafn",
						Name:              "abcd",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
			},
			HTTPStatus:    http.StatusOK,
			ProjectToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenDefaultCluster(),
				// add ssh keys
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "yafn",
						Clusters: []string{test.GenDefaultCluster().Name},
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
								Name:       test.GenDefaultProject().Name,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime.Add(time.Minute)),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "abcd",
						Clusters: []string{test.GenDefaultCluster().Name},
					},
				},
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ClusterToSync:   test.GenDefaultCluster().Name,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys", tc.ProjectToSync, tc.ClusterToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualKeys := test.NewSSHKeyV1SliceWrapper{}
			actualKeys.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedKeys := test.NewSSHKeyV1SliceWrapper(tc.ExpectedKeys)
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
		ClusterToSync          string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
		ExpectedSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:             "scenario 1: an ssh key that belongs to the given project is assigned to the cluster",
			SSHKeyID:         "key-c08aa5c7abf34504f18552846485267d-yafn",
			ExpectedResponse: `{"id":"key-c08aa5c7abf34504f18552846485267d-yafn","name":"","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"","publicKey":""}}`,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenDefaultCluster(),
				// add a ssh key
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
					Spec: kubermaticv1.SSHKeySpec{
						Clusters: []string{test.GenDefaultCluster().Name},
					},
				},
			),
			ClusterToSync: test.GenDefaultCluster().Name,
		},
		// scenario 2
		{
			Name:             "scenario 2: an ssh key that does not belong to the given project cannot be assigned to the cluster",
			SSHKeyID:         "key-c08aa5c7abf34504f18552846485267d-yafn",
			ExpectedResponse: `{"error":{"code":500,"message":"the given ssh key key-c08aa5c7abf34504f18552846485267d-yafn does not belong to the given project my-first-project (my-first-project-ID)"}}`,
			HTTPStatus:       http.StatusInternalServerError,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExpectedSSHKeys:  []*kubermaticv1.UserSSHKey{},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenDefaultCluster(),
				// add an ssh key
				&kubermaticv1.UserSSHKey{
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
			),
			ClusterToSync: test.GenDefaultCluster().Name,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/sshkeys/%s", tc.ProjectToSync, tc.ClusterToSync, tc.SSHKeyID), nil)
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)

			kubermaticClient := clientsSets.FakeKubermaticClient
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
								validatedActions++
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
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
		RewriteClusterID       bool
	}{
		// scenario 1
		{
			Name:                   "scenario 1: a cluster with invalid spec is rejected",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}, "version":""}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid cluster: invalid cloud spec \"Version\" is required but was not specified"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: cluster is created when valid spec and ssh key are passed",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.9.7","oidc":{}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add an ssh key
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       test.GenDefaultProject().Name,
							},
						},
					},
				},
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:                   "scenario 3: unable to create a cluster when the user doesn't belong to the project",
			Body:                   `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"fake":{},"dc":"fake-dc"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse:       `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			HTTPStatus:             http.StatusForbidden,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenUser("", "John", "john@acme.com")),
			ExistingAPIUser: func() *apiv1.User {
				defaultUser := test.GenDefaultAPIUser()
				defaultUser.Email = "john@acme.com"
				return defaultUser
			}(),
		},
		// scenario 4
		{
			Name:             "scenario 4: unable to create a cluster when project is not ready",
			Body:             `{"cluster":{"humanReadableName":"keen-snyder","version":"1.9.7","pause":false,"cloud":{"fake":{},"dc":"fake-dc"}},"sshKeys":["key-c08aa5c7abf34504f18552846485267d-yafn"]}`,
			ExpectedResponse: `{"error":{"code":503,"message":"Project is not initialized yet"}}`,
			HTTPStatus:       http.StatusServiceUnavailable,
			ExistingProject: func() *kubermaticv1.Project {
				project := test.GenDefaultProject()
				project.Status.Phase = kubermaticv1.ProjectInactive
				return project
			}(),
			ProjectToSync: test.GenDefaultProject().Name,
			ExistingKubermaticObjs: []runtime.Object{
				test.GenDefaultUser(),
				test.GenDefaultOwnerBinding(),
			},
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 5
		{
			Name:                   "scenario 5: openShift cluster is created",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","spec":{"version":"1.9.7","openshift":{"imagePullSecret": "some-secret"},"cloud":{"fake":{"token":"dummy_token"},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"openshift","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.9.7","oidc":{}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID:       true,
			HTTPStatus:             http.StatusCreated,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 6
		{
			Name:                   "scenario 6: openShift cluster is created with existing custom credential",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"fake","spec":{"version":"1.9.7","openshift":{"imagePullSecret": "some-secret"},"cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"openshift","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.9.7","oidc":{}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID:       true,
			HTTPStatus:             http.StatusCreated,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		// scenario 7
		{
			Name:                   "scenario 7: custom credential doesn't exist for Fake cloud provider",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"default","spec":{"version":"1.9.7","cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"invalid credentials: missing preset 'default' for the user 'bob@acme.com'"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 8: openShift cluster creation fails without imagePullSecret",
			Body:                   `{"cluster":{"name":"keen-snyder","type":"openshift","credential":"fake","spec":{"version":"1.9.7","cloud":{"fake":{},"dc":"fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":400,"message":"openshift clusters must be configured with an imagePullSecret"}}`,
			HTTPStatus:             http.StatusBadRequest,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 9a: rejected an attempt to create a cluster in email-restricted datacenter - legacy single domain restriction with requiredEmailDomains",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc"}}}}`,
			ExpectedResponse:       `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc\" not found"}}`,
			RewriteClusterID:       false,
			HTTPStatus:             http.StatusNotFound,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:                   "scenario 9b: rejected an attempt to create a cluster in email-restricted datacenter - domain array restriction with `requiredEmailDomains`",
			Body:                   `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc2"}}}}`,
			ExpectedResponse:       `{"error":{"code":404,"message":"datacenter \"restricted-fake-dc2\" not found"}}`,
			RewriteClusterID:       false,
			HTTPStatus:             http.StatusNotFound,
			ProjectToSync:          test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
		},
		{
			Name:             "scenario 10a: create a cluster in email-restricted datacenter, to which the user does have access - legacy single domain restriction with requiredEmailDomains",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"restricted-fake-dc","fake":{}},"version":"1.9.7","oidc":{}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			Name:             "scenario 10b: create a cluster in email-restricted datacenter, to which the user does have access - domain array restriction with `requiredEmailDomains`",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"restricted-fake-dc2"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"restricted-fake-dc2","fake":{}},"version":"1.9.7","oidc":{}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
		{
			Name:             "scenario 11: create a cluster in audit-logging-enforced datacenter, without explicitly enabling audit logging",
			Body:             `{"cluster":{"name":"keen-snyder","spec":{"version":"1.9.7","cloud":{"fake":{"token":"dummy_token"},"dc":"audited-dc"}}}}`,
			ExpectedResponse: `{"id":"%s","name":"keen-snyder","creationTimestamp":"0001-01-01T00:00:00Z","type":"kubernetes","spec":{"cloud":{"dc":"audited-dc","fake":{}},"version":"1.9.7","oidc":{},"auditLogging":{"enabled":true}},"status":{"version":"1.9.7","url":""}}`,
			RewriteClusterID: true,
			HTTPStatus:       http.StatusCreated,
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenUser(test.UserID2, test.UserName2, test.UserEmail2),
				test.GenBinding(test.GenDefaultProject().Name, test.UserEmail2, "editors"),
			),
			ExistingAPIUser: test.GenAPIUser(test.UserName2, test.UserEmail2),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters", tc.ProjectToSync), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
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
				actualCluster := &apiv1.Cluster{}
				err = json.Unmarshal(res.Body.Bytes(), actualCluster)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualCluster.ID)
			}

			test.CompareWithResult(t, res, expectedResponse)
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
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: get existing cluster health status",
			Body:             ``,
			ExpectedResponse: `{"apiserver":1,"scheduler":0,"controller":1,"machineController":0,"etcd":1,"cloudProviderInfrastructure":1,"userClusterControllerManager":1}`,
			HTTPStatus:       http.StatusOK,
			ClusterToGet:     "keen-snyder",
			ProjectToSync:    test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				// add a cluster
				test.GenCluster("clusterDefID", "clusterDef", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC)),
				// add another cluster
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Status.ExtendedHealth = kubermaticv1.ExtendedClusterHealth{

						Apiserver:                    kubermaticv1.HealthStatusUp,
						Scheduler:                    kubermaticv1.HealthStatusDown,
						Controller:                   kubermaticv1.HealthStatusUp,
						MachineController:            kubermaticv1.HealthStatusDown,
						Etcd:                         kubermaticv1.HealthStatusUp,
						CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
						UserClusterControllerManager: kubermaticv1.HealthStatusUp,
					}
					return cluster
				}(),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/health", tc.ProjectToSync, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestPatchCluster(t *testing.T) {
	t.Parallel()

	cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
	cluster.Spec.Cloud.DatacenterName = "us-central1"

	testcases := []struct {
		Name                      string
		Body                      string
		ExpectedResponse          string
		HTTPStatus                int
		cluster                   string
		project                   string
		ExistingAPIUser           *apiv1.User
		ExistingMachines          []*clusterv1alpha1.Machine
		ExistingKubermaticObjects []runtime.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: update the cluster version",
			Body:             `{"spec":{"version":"1.2.3"}}`,
			ExpectedResponse: `{"id":"keen-snyder","name":"clusterAbc","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.2.3","oidc":{}},"status":{"version":"1.2.3","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusOK,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = fakeDC
					return cluster
				}()),
		},
		// scenario 2
		{
			Name:                      "scenario 2: fail on invalid patch json",
			Body:                      `{"spec":{"cloud":{"dc":"dc1"`,
			ExpectedResponse:          `{"error":{"code":400,"message":"cannot patch cluster: Invalid JSON Patch"}}`,
			cluster:                   "keen-snyder",
			HTTPStatus:                http.StatusBadRequest,
			project:                   test.GenDefaultProject().Name,
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))),
		},
		// scenario 3
		{
			Name:             "scenario 3: tried to update cluser with older but compatible nodes",
			Body:             `{"spec":{"version":"9.11.3"}}`, // kubelet is 9.9.9, maximum compatible master is 9.11.x
			ExpectedResponse: `{"id":"keen-snyder","name":"clusterAbc","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"9.11.3","oidc":{}},"status":{"version":"9.11.3","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusOK,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = fakeDC
					return cluster
				}(),
			),
			ExistingMachines: []*clusterv1alpha1.Machine{
				test.GenTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				test.GenTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
		},
		// scenario 4
		{
			Name:             "scenario 4: tried to update cluser with old nodes",
			Body:             `{"spec":{"version":"9.12.3"}}`, // kubelet is 9.9.9, maximum compatible master is 9.11.x
			ExpectedResponse: `{"error":{"code":400,"message":"Cluster contains nodes running the following incompatible kubelet versions: [9.9.9]. Upgrade your nodes before you upgrade the cluster."}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusBadRequest,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = "us-central1"
					return cluster
				}(),
			),
			ExistingMachines: []*clusterv1alpha1.Machine{
				test.GenTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				test.GenTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
		},
		// scenario 5
		{
			Name:             "scenario 5: tried to downgrade cluser to version older than its nodes",
			Body:             `{"spec":{"version":"9.8.12"}}`, // kubelet is 9.9.9, cluster cannot be older
			ExpectedResponse: `{"error":{"code":400,"message":"Cluster contains nodes running the following incompatible kubelet versions: [9.9.9]. Upgrade your nodes before you upgrade the cluster."}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusBadRequest,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenDefaultAPIUser(),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = "us-central1"
					return cluster
				}(),
			),
			ExistingMachines: []*clusterv1alpha1.Machine{
				test.GenTestMachine("venus", `{"cloudProvider":"digitalocean","cloudProviderSpec":{"token":"dummy-token","region":"fra1","size":"2GB"},"operatingSystem":"ubuntu","containerRuntimeInfo":{"name":"docker","version":"1.13"},"operatingSystemSpec":{"distUpgradeOnBoot":true}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
				test.GenTestMachine("mars", `{"cloudProvider":"aws","cloudProviderSpec":{"token":"dummy-token","region":"eu-central-1","availabilityZone":"eu-central-1a","vpcId":"vpc-819f62e9","subnetId":"subnet-2bff4f43","instanceType":"t2.micro","diskSize":50}, "containerRuntimeInfo":{"name":"docker","version":"1.12"},"operatingSystem":"ubuntu", "operatingSystemSpec":{"distUpgradeOnBoot":false}}`, map[string]string{"md-id": "123", "some-other": "xyz"}, nil),
			},
		},
		// scenario 6
		{
			Name:             "scenario 6: the admin John can update Bob's cluster version",
			Body:             `{"spec":{"version":"1.2.3"}}`,
			ExpectedResponse: `{"id":"keen-snyder","name":"clusterAbc","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"fake-dc","fake":{}},"version":"1.2.3","oidc":{}},"status":{"version":"1.2.3","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusOK,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenAPIUser("John", "john@acme.com"),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = fakeDC
					return cluster
				}(), genUser("John", "john@acme.com", true)),
		},
		// scenario 7
		{
			Name:             "scenario 7: the regular user John can not update Bob's cluster version",
			Body:             `{"spec":{"version":"1.2.3"}}`,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			cluster:          "keen-snyder",
			HTTPStatus:       http.StatusForbidden,
			project:          test.GenDefaultProject().Name,
			ExistingAPIUser:  test.GenAPIUser("John", "john@acme.com"),
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(
				func() *kubermaticv1.Cluster {
					cluster := test.GenCluster("keen-snyder", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC))
					cluster.Spec.Cloud.DatacenterName = fakeDC
					return cluster
				}(), genUser("John", "john@acme.com", false)),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var machineObj []runtime.Object
			for _, existingMachine := range tc.ExistingMachines {
				machineObj = append(machineObj, existingMachine)
			}
			// test data
			req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", tc.project, tc.cluster), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []runtime.Object{}, machineObj, tc.ExistingKubermaticObjects, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			// act
			ep.ServeHTTP(res, req)

			// validate
			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
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
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets cluster with the given name that belongs to the given project",
			Body:             ``,
			ExpectedResponse: `{"id":"defClusterID","name":"defClusterName","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"FakeDatacenter","fake":{}},"version":"9.9.9","oidc":{}},"status":{"version":"9.9.9","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name:             "scenario 2: gets cluster for Openstack and no sensitive data (credentials) are returned",
			Body:             ``,
			ExpectedResponse: `{"id":"defClusterID","name":"defClusterName","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"OpenstackDatacenter","openstack":{"floatingIpPool":"floatingIPPool"}},"version":"9.9.9","oidc":{}},"status":{"version":"9.9.9","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genClusterWithOpenstack(test.GenDefaultCluster()),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 3
		{
			Name:             "scenario 3: the admin John can get Bob's cluster",
			Body:             ``,
			ExpectedResponse: `{"id":"defClusterID","name":"defClusterName","creationTimestamp":"2013-02-03T19:54:00Z","type":"kubernetes","spec":{"cloud":{"dc":"OpenstackDatacenter","openstack":{"floatingIpPool":"floatingIPPool"}},"version":"9.9.9","oidc":{}},"status":{"version":"9.9.9","url":"https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885"}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				genClusterWithOpenstack(test.GenDefaultCluster()),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
		// scenario 4
		{
			Name:             "scenario 4: the regular user John can not get Bob's cluster",
			Body:             ``,
			ExpectedResponse: `{"error":{"code":403,"message":"forbidden: \"john@acme.com\" doesn't belong to the given project = my-first-project-ID"}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusForbidden,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", false),
				genClusterWithOpenstack(test.GenDefaultCluster()),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s", test.ProjectName, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListClusters(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedClusters       []apiv1.Cluster
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list clusters that belong to the given project",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterAbcID",
						Name:              "clusterAbc",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterDefID",
						Name:              "clusterDef",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterOpenstackID",
						Name:              "clusterOpenstack",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "OpenstackDatacenter",
							Openstack: &kubermaticv1.OpenstackCloudSpec{
								FloatingIPPool: "floatingIPPool",
							},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				test.GenCluster("clusterDefID", "clusterDef", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC)),
				genClusterWithOpenstack(test.GenCluster("clusterOpenstackID", "clusterOpenstack", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC))),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name: "scenario 2: the admin John can list Bob's clusters",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterAbcID",
						Name:              "clusterAbc",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterDefID",
						Name:              "clusterDef",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterOpenstackID",
						Name:              "clusterOpenstack",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "OpenstackDatacenter",
							Openstack: &kubermaticv1.OpenstackCloudSpec{
								FloatingIPPool: "floatingIPPool",
							},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				test.GenCluster("clusterDefID", "clusterDef", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 01, 54, 0, 0, time.UTC)),
				genClusterWithOpenstack(test.GenCluster("clusterOpenstackID", "clusterOpenstack", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC))),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters", test.ProjectName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualClusters := test.NewClusterV1SliceWrapper{}
			actualClusters.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedClusters := test.NewClusterV1SliceWrapper(tc.ExpectedClusters)
			wrappedExpectedClusters.Sort()

			actualClusters.EqualOrDie(wrappedExpectedClusters, t)
		})
	}
}

func TestListClustersForProject(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		ExpectedClusters       []apiv1.Cluster
		HTTPStatus             int
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
	}{
		// scenario 1
		{
			Name: "scenario 1: list clusters that belong to the given project",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterAbcID",
						Name:              "clusterAbc",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterOpenstackID",
						Name:              "clusterOpenstack",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "OpenstackDatacenter",
							Openstack: &kubermaticv1.OpenstackCloudSpec{
								FloatingIPPool: "floatingIPPool",
							},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				genClusterWithOpenstack(test.GenCluster("clusterOpenstackID", "clusterOpenstack", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC))),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
		},
		// scenario 2
		{
			Name: "scenario 2: the admin John can list Bob's clusters in his project",
			ExpectedClusters: []apiv1.Cluster{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterAbcID",
						Name:              "clusterAbc",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "FakeDatacenter",
							Fake:           &kubermaticv1.FakeCloudSpec{},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "clusterOpenstackID",
						Name:              "clusterOpenstack",
						CreationTimestamp: apiv1.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC),
					},
					Spec: apiv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "OpenstackDatacenter",
							Openstack: &kubermaticv1.OpenstackCloudSpec{
								FloatingIPPool: "floatingIPPool",
							},
						},
						Version: *semver.NewSemverOrDie("9.9.9"),
					},
					Status: apiv1.ClusterStatus{
						Version: *semver.NewSemverOrDie("9.9.9"),
						URL:     "https://w225mx4z66.asia-east1-a-1.cloud.kubermatic.io:31885",
					},
					Type: "kubernetes",
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				genUser("John", "john@acme.com", true),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
				genClusterWithOpenstack(test.GenCluster("clusterOpenstackID", "clusterOpenstack", test.GenDefaultProject().Name, time.Date(2013, 02, 04, 03, 54, 0, 0, time.UTC))),
			),
			ExistingAPIUser: test.GenAPIUser("John", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/clusters", test.ProjectName), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualClusters := test.NewClusterV1SliceWrapper{}
			actualClusters.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedClusters := test.NewClusterV1SliceWrapper(tc.ExpectedClusters)
			wrappedExpectedClusters.Sort()

			actualClusters.EqualOrDie(wrappedExpectedClusters, t)
		})
	}
}

func TestRevokeClusterAdminTokenEndpoint(t *testing.T) {
	t.Parallel()
	// setup world view
	expectedResponse := "{}"
	kubermaticObjs := test.GenDefaultKubermaticObjects()
	tester := test.GenDefaultAPIUser()
	projectToSync := test.GenDefaultProject().Name
	cluster := test.GenDefaultCluster()
	kubermaticObjs = append(kubermaticObjs, cluster)
	ep, clientsSets, err := test.CreateTestEndpointAndGetClients(*tester, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObjs, nil, nil, hack.NewTestRouting)
	if err != nil {
		t.Fatalf("failed to create test endpoint due to %v", err)
	}

	// perform test
	res := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/token", projectToSync, cluster.Name), nil)
	ep.ServeHTTP(res, req)

	// check assertions
	test.CheckStatusCode(http.StatusOK, res, t)
	test.CompareWithResult(t, res, expectedResponse)
	updatedCluster := &kubermaticv1.Cluster{}
	if err := clientsSets.FakeClient.Get(context.Background(), types.NamespacedName{Name: test.DefaultClusterID}, updatedCluster); err != nil {
		t.Fatalf("failed to get cluster from fake client: %v", err)
	}
	updatedToken := updatedCluster.Address.AdminToken
	if err := kuberneteshelper.ValidateKubernetesToken(updatedToken); err != nil {
		t.Errorf("generated token '%s' is malformed: %v", updatedToken, err)
	}
	if updatedToken == cluster.Address.AdminToken {
		t.Errorf("generated token '%s' is exactly the same as the old one : %s", updatedToken, cluster.Address.AdminToken)
	}
}

func TestGetClusterEventsEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		HTTPStatus             int
		ExpectedResult         string
		ProjectIDToSync        string
		ClusterIDToSync        string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
		ExistingEvents         []*corev1.Event
		NodeDeploymentID       string
		QueryParams            string
	}{
		// scenario 1
		{
			Name:                   "scenario 1: list all events",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Cluster", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Cluster", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Cluster","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1},{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Cluster","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
		// scenario 2
		{
			Name:                   "scenario 2: list all warning events",
			QueryParams:            "?type=warning",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Cluster", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Cluster", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-2","creationTimestamp":"0001-01-01T00:00:00Z","message":"message killed","type":"Warning","involvedObject":{"type":"Cluster","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
		// scenario 3
		{
			Name:                   "scenario 3: list all normal events",
			QueryParams:            "?type=normal",
			HTTPStatus:             http.StatusOK,
			ClusterIDToSync:        test.GenDefaultCluster().Name,
			ProjectIDToSync:        test.GenDefaultProject().Name,
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(test.GenDefaultCluster()),
			ExistingAPIUser:        test.GenDefaultAPIUser(),
			ExistingEvents: []*corev1.Event{
				test.GenTestEvent("event-1", corev1.EventTypeNormal, "Started", "message started", "Cluster", "venus-1-machine"),
				test.GenTestEvent("event-2", corev1.EventTypeWarning, "Killed", "message killed", "Cluster", "venus-1-machine"),
			},
			ExpectedResult: `[{"name":"event-1","creationTimestamp":"0001-01-01T00:00:00Z","message":"message started","type":"Normal","involvedObject":{"type":"Cluster","namespace":"kube-system","name":"testMachine"},"lastTimestamp":"0001-01-01T00:00:00Z","count":1}]`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/events%s", tc.ProjectIDToSync, tc.ClusterIDToSync, tc.QueryParams), strings.NewReader(""))
			res := httptest.NewRecorder()
			kubermaticObj := make([]runtime.Object, 0)
			machineObj := make([]runtime.Object, 0)
			kubernetesObj := make([]runtime.Object, 0)
			for _, existingEvents := range tc.ExistingEvents {
				kubernetesObj = append(kubernetesObj, existingEvents)
			}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)

			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubernetesObj, machineObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResult)
		})
	}
}

func TestGetClusterMetrics(t *testing.T) {
	t.Parallel()
	cpuQuantity, err := resource.ParseQuantity("290")
	if err != nil {
		t.Fatal(err)
	}
	memoryQuantity, err := resource.ParseQuantity("687202304")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ClusterToGet           string
		ExistingAPIUser        *apiv1.User
		ExistingKubermaticObjs []runtime.Object
		ExistingNodes          []*corev1.Node
		ExistingPodMetrics     []*v1beta1.PodMetrics
		ExistingNodeMetrics    []*v1beta1.NodeMetrics
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets cluster metrics",
			Body:             ``,
			ExpectedResponse: `{"name":"defClusterID","controlPlane":{"memoryTotalBytes":1310,"cpuTotalMillicores":580000},"nodes":{"memoryTotalBytes":1310,"memoryAvailableBytes":1310,"memoryUsedPercentage":100,"cpuTotalMillicores":580000,"cpuAvailableMillicores":580000,"cpuUsedPercentage":100}}`,
			ClusterToGet:     test.GenDefaultCluster().Name,
			HTTPStatus:       http.StatusOK,
			ExistingNodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "venus"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "mars"}, Status: corev1.NodeStatus{Allocatable: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity}}},
			},
			ExistingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenCluster("clusterAbcID", "clusterAbc", test.GenDefaultProject().Name, time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC)),
			),
			ExistingAPIUser: test.GenDefaultAPIUser(),
			ExistingPodMetrics: []*v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "cluster-defClusterID"},
					Containers: []v1beta1.ContainerMetrics{
						{
							Name:  "c1-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
						{
							Name:  "c2-pod1",
							Usage: map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
						},
					},
				},
			},
			ExistingNodeMetrics: []*v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "venus"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "mars"},
					Usage:      map[corev1.ResourceName]resource.Quantity{"cpu": cpuQuantity, "memory": memoryQuantity},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			var kubermaticObj []runtime.Object
			for _, existingMetric := range tc.ExistingPodMetrics {
				kubernetesObj = append(kubernetesObj, existingMetric)
			}
			for _, existingMetric := range tc.ExistingNodeMetrics {
				kubernetesObj = append(kubernetesObj, existingMetric)
			}
			for _, node := range tc.ExistingNodes {
				kubeObj = append(kubeObj, node)
			}
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/metrics", test.ProjectName, tc.ClusterToGet), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()

			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestListNamespace(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name                   string
		expectedResponse       string
		httpStatus             int
		clusterToGet           string
		existingAPIUser        *apiv1.User
		existingKubermaticObjs []runtime.Object
		existingKubernrtesObjs []runtime.Object
	}{
		// scenario 1
		{
			name:             "scenario 1: get cluster namespaces",
			expectedResponse: `[{"name":"default"},{"name":"kube-admin"}]`,
			clusterToGet:     test.GenDefaultCluster().Name,
			httpStatus:       http.StatusOK,
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			existingKubernrtesObjs: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-admin"},
				},
			},
			existingAPIUser: test.GenDefaultAPIUser(),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var kubernetesObj []runtime.Object
			var kubeObj []runtime.Object
			kubeObj = append(kubeObj, tc.existingKubernrtesObjs...)
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/%s/namespaces", test.ProjectName, tc.clusterToGet), strings.NewReader(""))
			res := httptest.NewRecorder()
			var kubermaticObj []runtime.Object
			kubermaticObj = append(kubermaticObj, tc.existingKubermaticObjs...)
			ep, _, err := test.CreateTestEndpointAndGetClients(*tc.existingAPIUser, nil, kubeObj, kubernetesObj, kubermaticObj, nil, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.httpStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.httpStatus, res.Code, res.Body.String())
			}

			test.CompareWithResult(t, res, tc.expectedResponse)
		})
	}
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

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
