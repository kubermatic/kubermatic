package rancherserver

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// ServiceAccountCreator creates the service account for rancherserver
func ServiceAccountCreator() (string, reconciling.ServiceAccountCreator) {
	return resources.RancherServerServiceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
		sa.Labels = resources.BaseAppLabel(resources.RancherStatefulSetName, nil)
		return sa, nil
	}
}
