package cluster

import (
	"fmt"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/controller/resources"
	"github.com/kubermatic/kubermatic/api/extensions/etcd"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (cc *clusterController) healthyDep(ns, name string) (bool, error) {
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

	return dep.Status.AvailableReplicas == *dep.Spec.Replicas, nil
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

	e, ok := obj.(*etcd.Cluster)
	if !ok {
		return false, api.ErrInvalidType
	}

	return e.Status.Phase == etcd.ClusterPhaseRunning, nil
}
