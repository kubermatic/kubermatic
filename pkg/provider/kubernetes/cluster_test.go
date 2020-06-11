package kubernetes_test

import (
	"reflect"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		expectedError             string
		shareKubeconfig           bool
	}{
		{
			name:            "scenario 1, create kubernetes cluster",
			shareKubeconfig: false,
			workerName:      "test-kubernetes",
			userInfo:        &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:         genDefaultProject(),
			spec:            genClusterSpec("test-k8s"),
			clusterType:     "kubernetes",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedCluster: func() *kubermaticv1.Cluster {
				cluster := genCluster("test-k8s", "kubernetes", "my-first-project-ID", "test-kubernetes", "john@acme.com")
				cluster.ResourceVersion = "1"
				return cluster
			}(),
		},
		{
			name:            "scenario 2, create OpenShift cluster",
			shareKubeconfig: false,
			workerName:      "test-openshift",
			userInfo:        &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:         genDefaultProject(),
			spec:            genClusterSpec("test-openshift"),
			clusterType:     "openshift",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedCluster: func() *kubermaticv1.Cluster {
				cluster := genCluster("test-openshift", "openshift", "my-first-project-ID", "test-openshift", "john@acme.com")
				cluster.ResourceVersion = "1"
				return cluster
			}(),
		},
		{
			name:            "scenario 3, create kubernetes cluster when share kubeconfig is enabled and OIDC is set",
			shareKubeconfig: true,
			workerName:      "test-kubernetes",
			userInfo:        &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:         genDefaultProject(),
			spec: func() *kubermaticv1.ClusterSpec {
				spec := genClusterSpec("test-k8s")
				spec.OIDC = kubermaticv1.OIDCSettings{
					IssuerURL: "http://test",
					ClientID:  "test",
				}
				return spec
			}(),
			clusterType: "kubernetes",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedError: "can not set OIDC for the cluster when share config feature is enabled",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}

			// act
			target := kubernetes.NewClusterProvider(&restclient.Config{}, fakeImpersonationClient, nil, tc.workerName, nil, nil, nil, tc.shareKubeconfig)
			partialCluster := &kubermaticv1.Cluster{}
			partialCluster.Spec = *tc.spec
			if tc.clusterType == "openshift" {
				partialCluster.Annotations = map[string]string{
					"kubermatic.io/openshift": "true",
				}
			}
			if tc.expectedCluster != nil {
				partialCluster.Finalizers = tc.expectedCluster.Finalizers
			}

			cluster, err := target.New(tc.project, tc.userInfo, partialCluster)
			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error: %s", tc.expectedError)
				}
				if tc.expectedError != err.Error() {
					t.Fatalf("expected error: %s got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}

				// override autogenerated field
				cluster.Name = tc.expectedCluster.Name
				cluster.Status.NamespaceName = tc.expectedCluster.Status.NamespaceName

				if !reflect.DeepEqual(cluster, tc.expectedCluster) {
					t.Fatalf("%v", diff.ObjectGoPrintSideBySide(tc.expectedCluster, cluster))
				}
			}

		})
	}
}
