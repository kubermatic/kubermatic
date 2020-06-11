package coredns

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator returns a ConfigMap containing the config for the CoreDNS
func ConfigMapCreator() reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.CoreDNSConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.CoreDNSServiceName, nil)
			cm.Data["Corefile"] = `
      .:53 {
          errors
          health
          kubernetes cluster.local in-addr.arpa ip6.arpa {
             pods insecure
             upstream
             fallthrough in-addr.arpa ip6.arpa
          }
          prometheus :9153
          proxy . /etc/resolv.conf
          cache 30
          loop
          reload
          loadbalance
      }
      `

			return cm, nil
		}
	}
}
