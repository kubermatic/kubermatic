package cluster

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
)

func (cc *controller) healthyDep(dc, ns, name string, minReady int32) (bool, error) {
	informerGroup, err := cc.clientProvider.GetInformerGroup(dc)
	if err != nil {
		return false, fmt.Errorf("failed to get informer group for dc %q: %v", dc, err)
	}

	dep, err := informerGroup.DeploymentInformer.Lister().Deployments(ns).Get(name)
	if err != nil {
		return false, err
	}

	return dep.Status.AvailableReplicas == *dep.Spec.Replicas || dep.Status.AvailableReplicas >= minReady, nil
}

func (cc *controller) healthyEtcd(dc, ns, name string) (bool, error) {
	informerGroup, err := cc.clientProvider.GetInformerGroup(dc)
	if err != nil {
		return false, fmt.Errorf("failed to get informer group for dc %q: %v", dc, err)
	}

	etcd, err := informerGroup.EtcdClusterInformer.Lister().EtcdClusters(ns).Get(resources.EtcdClusterName)
	if err != nil {
		return false, err
	}

	return etcd.Status.Phase == etcdoperatorv1beta2.ClusterPhaseRunning, nil
}
