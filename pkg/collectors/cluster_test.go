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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

func TestClusterLabelsMetric(t *testing.T) {
	kubermaticFakeClient := fake.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster1",
				Labels: map[string]string{
					"UPPERCASE":            "123",
					"is-credential-preset": "true",
					"project-id":           "my-project",
				},
			},
		}).
		Build()

	registry := prometheus.NewRegistry()
	if err := registry.Register(newClusterCollector(kubermaticFakeClient)); err != nil {
		t.Fatal(err)
	}

	expected := `
# HELP kubermatic_cluster_labels Kubernetes labels on Cluster resources
# TYPE kubermatic_cluster_labels gauge
kubermatic_cluster_labels{label_is_credential_preset="true",label_project_id="my-project",label_uppercase="123",name="cluster1"} 0
`

	if err := testutil.CollectAndCompare(registry, strings.NewReader(expected), "kubermatic_cluster_labels"); err != nil {
		t.Fatal(err)
	}
}
