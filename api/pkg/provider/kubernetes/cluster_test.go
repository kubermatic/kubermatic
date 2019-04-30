package kubernetes_test

import (
	"testing"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"k8s.io/apimachinery/pkg/util/diff"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCreateCluster(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		workerName                string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		userInfo                  *provider.UserInfo
		spec                      *kubermaticv1.ClusterSpec
		clusterType               string
		expectedCluster           *kubermaticv1.Cluster
	}{
		{
			name:        "scenario 1, create kubernetes cluster",
			workerName:  "test-kubernetes",
			userInfo:    &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:     genDefaultProject(),
			spec:        genClusterSpec("test-k8s"),
			clusterType: "kubernetes",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedCluster: genCluster("test-k8s", "kubernetes", "my-first-project-ID", "test-kubernetes", "john@acme.com"),
		},
		{
			name:        "scenario 2, create OpenShift cluster",
			workerName:  "test-openshift",
			userInfo:    &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:     genDefaultProject(),
			spec:        genClusterSpec("test-openshift"),
			clusterType: "openshift",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedCluster: genCluster("test-openshift", "openshift", "my-first-project-ID", "test-openshift", "john@acme.com"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			impersonationClient, _, indexer, err := createFakeKubermaticClients(tc.existingKubermaticObjects)
			if err != nil {
				t.Fatalf("unable to create fake clients, err = %v", err)
			}

			clusterLister := kubermaticv1lister.NewClusterLister(indexer)

			// act
			target := kubernetes.NewClusterProvider(impersonationClient.CreateFakeImpersonatedClientSet, nil, clusterLister, tc.workerName, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			partialCluster := &provider.PartialCluster{}
			partialCluster.ClusterSpec = tc.spec
			if tc.clusterType == "openshift" {
				partialCluster.Annotations = map[string]string{
					"kubermatic.io/openshift": "true",
				}
			}

			cluster, err := target.New(tc.project, tc.userInfo, partialCluster)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			// override autogenerated field
			cluster.Name = tc.expectedCluster.Name
			cluster.Status.NamespaceName = tc.expectedCluster.Status.NamespaceName

			if !equality.Semantic.DeepEqual(cluster, tc.expectedCluster) {
				t.Fatalf("%v", diff.ObjectDiff(tc.expectedCluster, cluster))
			}
		})
	}
}
