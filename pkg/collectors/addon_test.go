/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cluster names may contain dashes, and the cluster label must carry the
// full name. Before the fix, the label was derived as the second dash
// separated token of the namespace, so cluster-acs-prod and
// cluster-acs-preprod both produced cluster="acs"; the identical label
// sets made the gather step fail the whole /metrics endpoint with
// HTTP 500. The dash-free name keeps its unchanged label value.
func TestAddonMetricsCarryFullClusterName(t *testing.T) {
	created := metav1.NewTime(time.Unix(1726000000, 0).UTC())

	kubermaticFakeClient := fake.
		NewClientBuilder().
		WithObjects(
			&kubermaticv1.Addon{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "csi",
					Namespace:         "cluster-acs-prod",
					CreationTimestamp: created,
				},
			},
			&kubermaticv1.Addon{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "csi",
					Namespace:         "cluster-acs-preprod",
					CreationTimestamp: created,
				},
			},
			&kubermaticv1.Addon{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "csi",
					Namespace:         "cluster-canary",
					CreationTimestamp: created,
				},
			},
		).
		Build()

	registry := prometheus.NewRegistry()
	MustRegisterAddonCollector(registry, kubermaticFakeClient)

	expected := `
# HELP kubermatic_addon_created Unix creation timestamp
# TYPE kubermatic_addon_created gauge
kubermatic_addon_created{addon="csi",cluster="acs-preprod"} 1.726e+09
kubermatic_addon_created{addon="csi",cluster="acs-prod"} 1.726e+09
kubermatic_addon_created{addon="csi",cluster="canary"} 1.726e+09
`

	if err := testutil.CollectAndCompare(registry, strings.NewReader(expected), "kubermatic_addon_created"); err != nil {
		t.Error(err)
	}

	// With no AddonResourcesCreated condition set, reconcile_failed is 1 for
	// each addon; the gather must stay error free with distinct label sets.
	expected = `
# HELP kubermatic_addon_reconcile_failed Reconcile is failing
# TYPE kubermatic_addon_reconcile_failed gauge
kubermatic_addon_reconcile_failed{addon="csi",cluster="acs-preprod"} 1
kubermatic_addon_reconcile_failed{addon="csi",cluster="acs-prod"} 1
kubermatic_addon_reconcile_failed{addon="csi",cluster="canary"} 1
`

	if err := testutil.CollectAndCompare(registry, strings.NewReader(expected), "kubermatic_addon_reconcile_failed"); err != nil {
		t.Error(err)
	}
}
