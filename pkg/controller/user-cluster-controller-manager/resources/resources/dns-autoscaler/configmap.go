package dnsautoscaler

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator returns a ConfigMap containing the config for the dns autoscaler.
func ConfigMapCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.DNSAutoscalerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.DNSAutoscalerDeploymentName, nil)
			cm.Data["linear"] = `
			{
				"min": 2,
				"coresPerReplica": 32,
				"nodesPerReplica": 4,
				"preventSinglePointFailure": true,
				"includeUnschedulableNodes": true
			}
      `

			return cm, nil
		}
	}
}
