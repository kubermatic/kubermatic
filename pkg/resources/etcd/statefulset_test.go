package etcd

import (
	"flag"
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	testhelper "github.com/kubermatic/kubermatic/api/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetEtcdCommand(t *testing.T) {

	tests := []struct {
		name                  string
		clusterName           string
		clusterNamespace      string
		migrate               bool
		enableCorruptionCheck bool
	}{
		{
			name:             "no-migration",
			clusterName:      "lg69pmx8wf",
			clusterNamespace: "cluster-lg69pmx8wf",
			migrate:          false,
		},
		{
			name:             "with-migration",
			clusterName:      "62m9k9tqlm",
			clusterNamespace: "cluster-62m9k9tqlm",
			migrate:          true,
		},
		{
			name:                  "with-corruption-flags",
			clusterName:           "lg69pmx8wf",
			clusterNamespace:      "cluster-lg69pmx8wf",
			migrate:               false,
			enableCorruptionCheck: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := getEtcdCommand(test.clusterName, test.clusterNamespace, test.migrate, test.enableCorruptionCheck)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(args) != 3 {
				t.Fatalf("got less arguments than expected. got %d expected %d", len(args), 3)
			}
			cmd := args[2]

			testhelper.CompareOutput(t, fmt.Sprintf("etcd-command-%s", test.name), cmd, *update, ".sh")
		})
	}
}

func TestGetImageTag(t *testing.T) {
	testCases := []struct {
		name           string
		cluster        *kubermaticv1.Cluster
		expectedResult string
	}{
		{
			name: "Openshift",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"kubermatic.io/openshift": "true"},
				},
				Spec: kubermaticv1.ClusterSpec{Version: *semver.NewSemverOrDie("4.17.18")},
			},
			expectedResult: imageTagV33,
		},
		{
			name: "Kubernetes 1.16",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{Version: *semver.NewSemverOrDie("1.16.0")},
			},
			expectedResult: imageTagV33,
		},
		{
			name: "Kubernetes 1.17",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{Version: *semver.NewSemverOrDie("1.17.0")},
			},
			expectedResult: imageTagV34,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if result := ImageTag(tc.cluster); result != tc.expectedResult {
				t.Fatalf("expected result %s but got result %s", tc.expectedResult, result)
			}
		})
	}
}
