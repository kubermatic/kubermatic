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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// lokiConfigMapName is the ConfigMap the upstream Loki chart renders the
	// Loki configuration into (both chart 5.x shipped with KKP 2.30 and 7.x).
	lokiConfigMapName = "loki"
	lokiConfigKey     = "config.yaml"

	lokiTSDBStore = "tsdb"
)

var lokiSchemaConfigsPath = yamled.Path{"loki", "loki", "schemaConfig", "configs"}

type lokiPeriodConfig struct {
	From        string `yaml:"from"`
	Store       string `yaml:"store"`
	ObjectStore string `yaml:"object_store"`
	Schema      string `yaml:"schema"`
	Index       struct {
		Prefix string `yaml:"prefix"`
		Period string `yaml:"period"`
	} `yaml:"index"`
}

// ensureLokiSchemaBoundary pins the start date of the tsdb schema period.
//
// A schema period must never start before the day the Loki instance actually
// began writing with it, otherwise Loki looks up the index of older days in the
// wrong store and all data written before is unreadable. The chart ships a fixed
// date, which is only correct for fresh installations. This function therefore
// overrides it based on the currently deployed Loki configuration:
//
//   - no Loki deployed yet: the chart default is kept, there is no data to read.
//   - Loki deployed without a tsdb period (upgrade from Loki 2.x): the tsdb
//     period starts tomorrow (UTC), so the rest of today is still written to the
//     old store and no data becomes unreadable.
//   - Loki deployed with a tsdb period: its start date is kept, as changing it
//     would make already written data unreadable.
//
// If the user configured the schema explicitly, it is left untouched.
func ensureLokiSchemaBoundary(ctx context.Context, logger logrus.FieldLogger, kubeClient ctrlruntimeclient.Client, chartDirectory string, values *yamled.Document, now time.Time) error {
	if values.Has(lokiSchemaConfigsPath) {
		logger.Info("Using Loki schema configuration from the Helm values.")
		return nil
	}

	configs, err := loadChartLokiSchemaConfigs(chartDirectory)
	if err != nil {
		return fmt.Errorf("failed to load default schema configuration from chart: %w", err)
	}

	tsdbIndex := findLokiTSDBPeriod(configs)
	if tsdbIndex < 0 {
		return nil
	}

	deployed, err := getDeployedLokiSchemaConfigs(ctx, kubeClient)
	if err != nil {
		return fmt.Errorf("failed to determine deployed Loki schema configuration: %w", err)
	}

	// fresh installation
	if deployed == nil {
		return nil
	}

	var from string
	if deployedIndex := findLokiTSDBPeriod(deployed); deployedIndex >= 0 {
		from = deployed[deployedIndex].From
	} else {
		from = now.UTC().AddDate(0, 0, 1).Format(time.DateOnly)
		logger.Infof("Upgrading Loki to the tsdb index store, new schema period starts at %s.", from)
	}

	configs[tsdbIndex].From = from
	if !values.Set(lokiSchemaConfigsPath, configs) {
		return fmt.Errorf("failed to set %v in Helm values", lokiSchemaConfigsPath)
	}

	return nil
}

func loadChartLokiSchemaConfigs(chartDirectory string) ([]lokiPeriodConfig, error) {
	f, err := os.Open(filepath.Join(chartDirectory, "values.yaml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := yamled.Load(f)
	if err != nil {
		return nil, err
	}

	var configs []lokiPeriodConfig
	if err := doc.DecodeAtPath(lokiSchemaConfigsPath, &configs); err != nil {
		return nil, err
	}

	return configs, nil
}

// getDeployedLokiSchemaConfigs returns nil if Loki is not deployed.
func getDeployedLokiSchemaConfigs(ctx context.Context, kubeClient ctrlruntimeclient.Client) ([]lokiPeriodConfig, error) {
	cm := &corev1.ConfigMap{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: LokiNamespace, Name: lokiConfigMapName}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	config := struct {
		SchemaConfig struct {
			Configs []lokiPeriodConfig `yaml:"configs"`
		} `yaml:"schema_config"`
	}{}

	if err := yaml.NewDecoder(strings.NewReader(cm.Data[lokiConfigKey])).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse %s in ConfigMap %s/%s: %w", lokiConfigKey, LokiNamespace, lokiConfigMapName, err)
	}

	if len(config.SchemaConfig.Configs) == 0 {
		return nil, fmt.Errorf("no schema_config found in ConfigMap %s/%s", LokiNamespace, lokiConfigMapName)
	}

	return config.SchemaConfig.Configs, nil
}

func findLokiTSDBPeriod(configs []lokiPeriodConfig) int {
	for i, config := range configs {
		if config.Store == lokiTSDBStore {
			return i
		}
	}

	return -1
}
