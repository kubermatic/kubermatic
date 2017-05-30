package cluster

import (
	"github.com/kubermatic/api/pkg/extensions/etcd"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

const (
	healthBar = 0.9
)

func (cc *clusterController) healthyDep(dep *v1beta1.Deployment) (bool, error) {
	replicas := dep.Spec.Replicas

	l := labels.SelectorFromSet(labels.Set(dep.Spec.Selector.MatchLabels))
	var pods []v1.Pod
	for _, m := range cc.podStore.List() {
		pod := m.(*v1.Pod)
		if l.Matches(labels.Set(pod.Labels)) {
			pods = append(pods, *pod)
		}
	}

	healthyPods := 0

	for _, p := range pods {
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Status.Phase != v1.PodRunning {
			continue
		}
		for _, c := range p.Status.Conditions {
			if c.Type == v1.PodReady && c.Status == v1.ConditionTrue {
				healthyPods++
				break
			}
		}
	}

	if float64(healthyPods) < healthBar*float64(*replicas) {
		return false, nil
	}

	return true, nil
}

func (cc *clusterController) healthyEtcd(etcd *etcd.Cluster) (bool, error) {

	//Ensure the etcd quorum
	if etcd.Spec.Size/2+1 >= etcd.Status.Size {
		return false, nil
	}

	return true, nil
}
