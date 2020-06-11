package etcd

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type pdbData interface {
	Cluster() *kubermaticv1.Cluster
}

// PodDisruptionBudgetCreator returns a func to create/update the etcd PodDisruptionBudget
func PodDisruptionBudgetCreator(data pdbData) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return resources.EtcdPodDisruptionBudgetName, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
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

}
