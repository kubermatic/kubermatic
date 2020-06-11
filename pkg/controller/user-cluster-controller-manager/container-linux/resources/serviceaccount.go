package resources

import (
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	ServiceAccountName = "container-linux-update-operator"
)

func ServiceAccountCreator() reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return ServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}
