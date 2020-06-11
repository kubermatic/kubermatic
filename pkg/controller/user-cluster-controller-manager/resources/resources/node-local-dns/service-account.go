package nodelocaldns

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountCreator creates the service account for Node Local DNS cache
func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return resources.NodeLocalDNSServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			if sa.Labels == nil {
				sa.Labels = map[string]string{}
			}
			sa.Labels["kubernetes.io/cluster-service"] = "true"
			sa.Labels[addonManagerModeKey] = reconcileModeValue

			return sa, nil
		}
	}
}
