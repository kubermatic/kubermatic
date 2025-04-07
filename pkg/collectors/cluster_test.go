/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package collectors

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterLabelsMetric(t *testing.T) {
	kubermaticFakeClient := fake.
		NewClientBuilder().
		WithObjects(
			&kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster1",
					Labels: map[string]string{
						"UPPERCASE":            "123",
						"is-credential-preset": "true",
						"project-id":           "my-project",
					},
				},
				Status: kubermaticv1.ClusterStatus{
					UserEmail: "test@example.com",
				},
			},
			&kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster2",
				},
				Status: kubermaticv1.ClusterStatus{
					UserEmail: "valid-user@example.com",
				},
			},
			&kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "username",
				},
				Spec: kubermaticv1.UserSpec{
					Email: "valid-user@example.com",
				},
			},
		).
		Build()

	registry := prometheus.NewRegistry()
	if err := registry.Register(newClusterCollector(kubermaticFakeClient)); err != nil {
		t.Fatal(err)
	}

	expected := `
# HELP kubermatic_cluster_labels Kubernetes labels on Cluster resources
# TYPE kubermatic_cluster_labels gauge
kubermatic_cluster_labels{label_is_credential_preset="",label_project_id="",label_uppercase="",name="cluster2"} 1
kubermatic_cluster_labels{label_is_credential_preset="true",label_project_id="my-project",label_uppercase="123",name="cluster1"} 1
`

	if err := testutil.CollectAndCompare(registry, strings.NewReader(expected), "kubermatic_cluster_labels"); err != nil {
		t.Error(err)
	}

	expected = `
# HELP kubermatic_cluster_owner Synthetic metric that maps clusters to their owners
# TYPE kubermatic_cluster_owner gauge
kubermatic_cluster_owner{cluster_name="cluster1",user="test@example.com"} 1
kubermatic_cluster_owner{cluster_name="cluster2",user="username"} 1
`

	if err := testutil.CollectAndCompare(registry, strings.NewReader(expected), "kubermatic_cluster_owner"); err != nil {
		t.Error(err)
	}
}
