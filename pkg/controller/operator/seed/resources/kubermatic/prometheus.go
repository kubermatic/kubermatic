package kubermatic

import (
	operatorv1alpha1 "github.com/kubermatic/kubermatic/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

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
