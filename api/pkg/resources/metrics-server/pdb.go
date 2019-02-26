package metricsserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDisruptionBudgetCreator returns a func to create/update the metrics-server PodDisruptionBudget
func PodDisruptionBudgetCreator() resources.PodDisruptionBudgetCreator {
	return func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
		pdb.Name = resources.MetricsServerPodDisruptionBudgetName

		maxUnavailable := intstr.FromInt(1)
		pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabel(name, nil),
			},
			MaxUnavailable: &maxUnavailable,
		}

		return pdb, nil
	}
}
