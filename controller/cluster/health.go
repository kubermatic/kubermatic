package cluster

import (
	"github.com/kubermatic/api"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
)

const (
	healthBar = 0.9
)

func (cc *clusterController) healthyRC(rc *kapi.ReplicationController) (bool, error) {
	replicas := rc.Spec.Replicas
	pods, err := cc.podStore.List(labels.SelectorFromSet(labels.Set(rc.Spec.Selector)))
	if err != nil {
		return false, err
	}

	healthyPods := 0
	for _, p := range pods {
		if p.DeletionTimestamp != nil {
			continue
		}
		if p.Status.Phase != kapi.PodRunning {
			continue
		}
		for _, c := range p.Status.Conditions {
			if c.Type != kapi.PodReady || c.Status != kapi.ConditionTrue {
				continue
			}
		}
		healthyPods++
	}

	if float64(healthyPods) < healthBar*float64(replicas) {
		return false, nil
	}

	return true, nil
}

func overallHealthy(h *api.ClusterHealth) bool {
	if h == nil {
		return false
	}
	healthy := true
	healthy = healthy && h.Apiserver
	healthy = healthy && h.Controller
	healthy = healthy && h.Scheduler
	for _, eh := range h.Etcd {
		healthy = healthy && eh
	}

	return healthy
}
