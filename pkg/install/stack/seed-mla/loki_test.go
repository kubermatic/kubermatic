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

package seedmla

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const lokiChartDirectory = "../../../../charts/logging/loki"

// rendered by the Loki chart shipped with KKP 2.30 (Loki 2.9).
const lokiConfigBoltDBOnly = `
schema_config:
  configs:
  - from: "2021-07-01"
    index:
      period: 24h
      prefix: index_
    object_store: filesystem
    schema: v11
    store: boltdb-shipper
  - from: "2022-01-11"
    index:
      period: 24h
      prefix: loki_index_
    object_store: filesystem
    schema: v12
    store: boltdb-shipper
`

const lokiConfigWithTSDB = `
schema_config:
  configs:
  - from: "2022-01-11"
    index:
      period: 24h
      prefix: loki_index_
    object_store: filesystem
    schema: v12
    store: boltdb-shipper
  - from: "2026-09-28"
    index:
      period: 24h
      prefix: index_
    object_store: filesystem
    schema: v13
    store: tsdb
`

func TestEnsureLokiSchemaBoundary(t *testing.T) {
	now := time.Date(2026, 10, 27, 23, 30, 0, 0, time.FixedZone("UTC-2", -2*60*60))

	testcases := []struct {
		name         string
		values       string
		deployed     string
		expectedTSDB string
		userProvided bool
	}{
		{
			name:     "fresh installation keeps the chart default",
			values:   "loki: {}",
			deployed: "",
		},
		{
			name:         "upgrade from Loki 2.x starts the tsdb period tomorrow (UTC)",
			values:       "loki: {}",
			deployed:     lokiConfigBoltDBOnly,
			expectedTSDB: "2026-10-29",
		},
		{
			name:         "existing tsdb period is kept",
			values:       "loki: {}",
			deployed:     lokiConfigWithTSDB,
			expectedTSDB: "2026-09-28",
		},
		{
			name: "user-provided schema is not touched",
			values: `
loki:
  loki:
    schemaConfig:
      configs:
      - from: "2026-10-01"
        store: tsdb
`,
			deployed:     lokiConfigBoltDBOnly,
			expectedTSDB: "2026-10-01",
			userProvided: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := yamled.Load(strings.NewReader(tc.values))
			if err != nil {
				t.Fatalf("failed to load values: %v", err)
			}

			objects := []ctrlruntimeclient.Object{}
			if tc.deployed != "" {
				objects = append(objects, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Namespace: LokiNamespace, Name: lokiConfigMapName},
					Data:       map[string]string{lokiConfigKey: tc.deployed},
				})
			}

			kubeClient := fake.NewClientBuilder().WithObjects(objects...).Build()

			if err := ensureLokiSchemaBoundary(context.Background(), logrus.New(), kubeClient, lokiChartDirectory, values, now); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectedTSDB == "" {
				if values.Has(lokiSchemaConfigsPath) {
					t.Fatalf("expected schema configuration not to be set, but it is")
				}

				return
			}

			var configs []lokiPeriodConfig
			if err := values.DecodeAtPath(lokiSchemaConfigsPath, &configs); err != nil {
				t.Fatalf("failed to decode schema configuration: %v", err)
			}

			idx := findLokiTSDBPeriod(configs)
			if idx < 0 {
				t.Fatalf("no tsdb period found in %+v", configs)
			}

			if configs[idx].From != tc.expectedTSDB {
				t.Fatalf("expected tsdb period to start at %q, got %q", tc.expectedTSDB, configs[idx].From)
			}

			// the boltdb-shipper period from the chart must be retained
			if !tc.userProvided && (idx == 0 || configs[0].From != "2022-01-11" || configs[0].Index.Prefix != "loki_index_") {
				t.Fatalf("expected boltdb-shipper period from chart defaults to be retained, got %+v", configs)
			}
		})
	}
}

func TestEnsureLokiSchemaBoundaryFailsOnUnparsableConfig(t *testing.T) {
	values, err := yamled.Load(strings.NewReader("loki: {}"))
	if err != nil {
		t.Fatalf("failed to load values: %v", err)
	}

	kubeClient := fake.NewClientBuilder().WithObjects(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: LokiNamespace, Name: lokiConfigMapName},
		Data:       map[string]string{lokiConfigKey: "auth_enabled: false"},
	}).Build()

	if err := ensureLokiSchemaBoundary(context.Background(), logrus.New(), kubeClient, lokiChartDirectory, values, time.Now()); err == nil {
		t.Fatal("expected an error, got none")
	}
}
