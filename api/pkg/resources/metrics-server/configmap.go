package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator returns a function to create the ConfigMap containing the cloud-config
func ConfigMapCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.MetricsServerConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Labels = resources.BaseAppLabel(name, nil)
			cm.Data["config.yaml"] = cfg

			return cm, nil
		}
	}
}

const cfg = `rules:
resourceRules:
  cpu:
    containerQuery: sum(rate(container_cpu_usage_seconds_total{<<.LabelMatchers>>}[5m]))
      by (<<.GroupBy>>)
    nodeQuery: sum(rate(container_cpu_usage_seconds_total{<<.LabelMatchers>>, id='/'}[5m]))
      by (<<.GroupBy>>)
    resources:
      overrides:
        instance:
          resource: node
        namespace:
          resource: namespace
        pod_name:
          resource: pod
    containerLabel: container_name
  memory:
    containerQuery: sum(container_memory_working_set_bytes{<<.LabelMatchers>>}) by
      (<<.GroupBy>>)
    nodeQuery: sum(container_memory_working_set_bytes{<<.LabelMatchers>>,id='/'})
      by (<<.GroupBy>>)
    resources:
      overrides:
        instance:
          resource: node
        namespace:
          resource: namespace
        pod_name:
          resource: pod
    containerLabel: container_name
  window: 5m
`
