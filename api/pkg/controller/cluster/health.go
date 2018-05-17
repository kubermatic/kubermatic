package cluster

import (
	"k8s.io/apimachinery/pkg/api/errors"
)

func (cc *Controller) healthyDeployment(ns, name string) (bool, error) {
	dep, err := cc.deploymentLister.Deployments(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return dep.Status.ReadyReplicas == *dep.Spec.Replicas && dep.Status.UpdatedReplicas == *dep.Spec.Replicas, nil
}

func (cc *Controller) healthyStatefulSet(ns, name string) (bool, error) {
	set, err := cc.statefulSetLister.StatefulSets(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return set.Status.ReadyReplicas == *set.Spec.Replicas && set.Status.CurrentReplicas == *set.Spec.Replicas, nil
}
