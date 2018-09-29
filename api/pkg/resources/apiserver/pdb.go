package apiserver

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDisruptionBudget returns the apiserver PodDisruptionBudget
func PodDisruptionBudget(data *resources.TemplateData, existing *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
	var pdb *policyv1beta1.PodDisruptionBudget
	if existing != nil {
		pdb = existing
	} else {
		pdb = &policyv1beta1.PodDisruptionBudget{}
	}

	pdb.Name = resources.ApiserverPodDisruptionBudgetName
	pdb.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	maxUnavailable := intstr.FromInt(1)
	pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: resources.BaseAppLabel(name, nil),
		},
		// we can only specify maxUnavailable as minAvailable would block in case we have replicas=1. See https://github.com/kubernetes/kubernetes/issues/66811
		MaxUnavailable: &maxUnavailable,
	}

	return pdb, nil
}
