package kubermatic

import (
	"encoding/json"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"
)

const (
	clusterNamespacePrometheusScrapingConfigMapName = "clusterns-prometheus-scraping-configs"
	clusterNamespacePrometheusRulesConfigMapName    = "clusterns-prometheus-rules"
)

func ClusterNamespacePrometheusScrapingConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	if len(cfg.Spec.SeedController.Monitoring.CustomScrapingConfigs) == 0 {
		return nil
	}

	return func() (string, reconciling.ConfigMapCreator) {
		return clusterNamespacePrometheusScrapingConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			configs := make([]interface{}, len(cfg.Spec.SeedController.Monitoring.CustomScrapingConfigs))
			for n, c := range cfg.Spec.SeedController.Monitoring.CustomScrapingConfigs {
				var config interface{}

				if err := json.Unmarshal(c.Raw, &config); err != nil {
					return nil, fmt.Errorf("Prometheus scraping rule %d is invalid: %v", n+1, err)
				}

				configs[n] = config
			}

			marshalled, err := yaml.Marshal(configs)
			if err != nil {
				return nil, fmt.Errorf("failed to encode Prometheus scraping rules as YAML: %v", err)
			}

			c.Data["_custom-scraping-configs.yaml"] = string(marshalled)

			return c, nil
		}
	}
}

func ClusterNamespacePrometheusRulesConfigMapCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedConfigMapCreatorGetter {
	if cfg.Spec.SeedController.Monitoring.CustomRules.Size() == 0 {
		return nil
	}

	return func() (string, reconciling.ConfigMapCreator) {
		return clusterNamespacePrometheusRulesConfigMapName, func(c *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if c.Data == nil {
				c.Data = make(map[string]string)
			}

			var rules interface{}
			if err := json.Unmarshal(cfg.Spec.SeedController.Monitoring.CustomRules.Raw, &rules); err != nil {
				return nil, fmt.Errorf("Prometheus rules are invalid: %v", err)
			}

			marshalled, err := yaml.Marshal(rules)
			if err != nil {
				return nil, fmt.Errorf("failed to encode Prometheus rules as YAML: %v", err)
			}

			c.Data["_customrules.yaml"] = string(marshalled)

			return c, nil
		}
	}
}
