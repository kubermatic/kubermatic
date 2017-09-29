package cluster

import (
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
)

func (cc *controller) healthyDep(ns, name string, minReady int32) (bool, error) {
	dep, err := cc.seedInformerGroup.DeploymentInformer.Lister().Deployments(ns).Get(name)
	if err != nil {
		return false, err
	}

	return dep.Status.AvailableReplicas == *dep.Spec.Replicas || dep.Status.AvailableReplicas >= minReady, nil
}

func (cc *controller) healthyEtcd(ns, name string) (bool, error) {
	etcd, err := cc.seedInformerGroup.EtcdClusterInformer.Lister().EtcdClusters(ns).Get(resources.EtcdClusterName)
	if err != nil {
		return false, err
	}

	return etcd.Status.Phase == etcdoperatorv1beta2.ClusterPhaseRunning, nil
}
