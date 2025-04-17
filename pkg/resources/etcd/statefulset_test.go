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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	testhelper "k8c.io/kubermatic/v2/pkg/test"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGetEtcdCommand(t *testing.T) {
	tests := []struct {
		name                  string
		cluster               *kubermaticv1.Cluster
		enableCorruptionCheck bool
		launcherEnabled       bool
		quotaBackendBytes     int64
		expectedArgs          int
	}{
		{
			name: "with-launcher",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "62m9k9tqlm",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-62m9k9tqlm",
				},
			},
			launcherEnabled: true,
			expectedArgs:    12,
		},
		{
			name: "without-launcher",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "62m9k9tqlm",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-62m9k9tqlm",
				},
			},
			launcherEnabled: false,
			expectedArgs:    30,
		},
		{
			name: "with-launcher-and-quota-backend-bytes",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "62m9k9tqlm",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-62m9k9tqlm",
				},
			},
			launcherEnabled:       true,
			quotaBackendBytes:     5959,
			expectedArgs:          14,
		},
		{
			name: "without-launcher-and-quota-backend-bytes",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "62m9k9tqlm",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-62m9k9tqlm",
				},
			},
			launcherEnabled:       false,
			quotaBackendBytes:     5959,
			expectedArgs:          32,
		},
		{
			name: "with-launcher-and-with-corruption-flags",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "lg69pmx8wf",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-lg69pmx8wf",
				},
			},
			enableCorruptionCheck: true,
			launcherEnabled:       true,
			expectedArgs:          15,
		},
		{
			name: "without-launcher-and-with-corruption-flags",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "lg69pmx8wf",
				},
				Status: kubermaticv1.ClusterStatus{
					NamespaceName: "cluster-lg69pmx8wf",
				},
			},
			enableCorruptionCheck: true,
			launcherEnabled:       false,
			expectedArgs:          33,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := getEtcdCommand(test.cluster, test.enableCorruptionCheck, test.launcherEnabled, test.quotaBackendBytes)

			if len(args) != test.expectedArgs {
				t.Fatalf("got less/more arguments than expected. got %d expected %d: %s", len(args), test.expectedArgs, strings.Join(args, " "))
			}
			cmd := strings.Join(args, " ")

			testhelper.CompareOutput(t, fmt.Sprintf("etcd-command-%s", test.name), cmd, *update, ".sh")
		})
	}
}
