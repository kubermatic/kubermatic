package cluster

import (
	"github.com/kubermatic/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/labels"
)

const (
	healthBar = 0.9
)

func (cc *clusterController) healthyDep(dep *v1beta1.Deployment) (bool, error) {
	replicas := dep.Spec.Replicas
	pods, err := cc.podStore.List(labels.SelectorFromSet(labels.Set(dep.Spec.Selector.MatchLabels)))
	if err != nil {
		return false, err
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
