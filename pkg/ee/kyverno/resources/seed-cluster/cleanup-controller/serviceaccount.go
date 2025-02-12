package resources

import (
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	name = "kyverno-cleanup-controller"
)

// ServiceAccountReconciler returns the function to create and update the Kyverno cleanup controller service account.
func ServiceAccountReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return name, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Labels = map[string]string{
				"app.kubernetes.io/component": "cleanup-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.13.2",
			}
			return sa, nil
		}
	}
}
