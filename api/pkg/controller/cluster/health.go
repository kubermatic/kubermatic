package cluster

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"

	"k8s.io/api/extensions/v1beta1"
)

func (cc *clusterController) healthyDep(ns, name string, minReady int32) (bool, error) {
	key := fmt.Sprintf("%s/%s", ns, name)
	obj, exists, err := cc.depStore.GetByKey(key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, api.ErrNotFound
	}

	dep, ok := obj.(*v1beta1.Deployment)
	if !ok {
		return false, api.ErrInvalidType
	}

	return dep.Status.AvailableReplicas == *dep.Spec.Replicas || dep.Status.AvailableReplicas >= minReady, nil
}

func (cc *clusterController) healthyEtcd(ns, name string) (bool, error) {
	key := fmt.Sprintf("%s/%s", ns, resources.EtcdClusterName)
	obj, exists, err := cc.etcdClusterStore.GetByKey(key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, api.ErrNotFound
	}

	e, ok := obj.(*etcdoperatorv1beta2.EtcdCluster)
	if !ok {
		return false, api.ErrInvalidType
	}

	return e.Status.Phase == etcdoperatorv1beta2.ClusterPhaseRunning, nil
}
