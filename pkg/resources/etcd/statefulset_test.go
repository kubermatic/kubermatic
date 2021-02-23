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

package etcd

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	testhelper "k8c.io/kubermatic/v2/pkg/test"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetEtcdCommand(t *testing.T) {

	tests := []struct {
		name                  string
		clusterName           string
		clusterNamespace      string
		enableCorruptionCheck bool
		launcherEnabled       bool
		expectedArgs          int
	}{
		{
			name:             "with-launcher",
			clusterName:      "62m9k9tqlm",
			clusterNamespace: "cluster-62m9k9tqlm",
			launcherEnabled:  true,
			expectedArgs:     13,
		},
		{
			name:                  "with-corruption-flags",
			clusterName:           "lg69pmx8wf",
			clusterNamespace:      "cluster-lg69pmx8wf",
			enableCorruptionCheck: true,
			launcherEnabled:       false,
			expectedArgs:          3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := getEtcdCommand(test.clusterName, test.clusterNamespace, test.enableCorruptionCheck, test.launcherEnabled)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(args) != test.expectedArgs {
				t.Fatalf("got less/more arguments than expected. got %d expected %d", len(args), test.expectedArgs)
			}
			cmd := strings.Join(args, " ")
			if !test.launcherEnabled {
				cmd = args[2]
			}

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
			name: "Kubernetes 1.17",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{Version: *semver.NewSemverOrDie("1.17.0")},
			},
			expectedResult: etcdImageTagV34,
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
