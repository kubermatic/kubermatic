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

package kubermatic

import (
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	clusterNamespacePrometheusScrapingConfigsConfigMapName = "clusterns-prometheus-scraping-configs"
	clusterNamespacePrometheusRulesConfigMapName           = "clusterns-prometheus-rules"

	clusterNamespacePrometheusScrapingConfigsKey = "_custom-scraping-configs.yaml"
	clusterNamespacePrometheusRulesKey           = "_customrules.yaml"
)

func ClusterNamespacePrometheusScrapingConfigsConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	if len(cfg.Spec.UserCluster.Monitoring.CustomScrapingConfigs) == 0 {
		return nil
	}

	return func() (string, reconciling.ConfigMapCreator) {
		return clusterNamespacePrometheusScrapingConfigsConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data[clusterNamespacePrometheusScrapingConfigsKey] = cfg.Spec.UserCluster.Monitoring.CustomScrapingConfigs

			return c, nil
		}
	}
}

func ClusterNamespacePrometheusRulesConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	if len(cfg.Spec.UserCluster.Monitoring.CustomRules) == 0 {
		return nil
	}

	return func() (string, reconciling.ConfigMapCreator) {
		return clusterNamespacePrometheusRulesConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			c.Data[clusterNamespacePrometheusRulesKey] = cfg.Spec.UserCluster.Monitoring.CustomRules

			return c, nil
		}
	}
}
