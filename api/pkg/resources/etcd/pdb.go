package etcd

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/util/intstr"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDisruptionBudgetCreator returns a func to create/update the etcd PodDisruptionBudget
func PodDisruptionBudgetCreator(data *resources.TemplateData) resources.PodDisruptionBudgetCreator {
	return func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
		pdb.Name = resources.EtcdPodDisruptionBudgetName
		pdb.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		minAvailable := intstr.FromInt((resources.EtcdClusterSize / 2) + 1)
		pdb.Spec = policyv1beta1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getBasePodLabels(data.Cluster()),
			},
			MinAvailable: &minAvailable,
		}

		return pdb, nil
	}
}
