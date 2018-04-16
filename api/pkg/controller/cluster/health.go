package cluster

import (
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (cc *Controller) healthyDeployment(ns, name string, minReady int32) (bool, error) {
	dep, err := cc.DeploymentLister.Deployments(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return dep.Status.AvailableReplicas == *dep.Spec.Replicas || dep.Status.AvailableReplicas >= minReady, nil
}

func (cc *Controller) healthyEtcd(ns, name string) (bool, error) {
	etcd, err := cc.EtcdClusterLister.EtcdClusters(ns).Get(resources.EtcdClusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return etcd.Status.Phase == etcdoperatorv1beta2.ClusterPhaseRunning, nil
}
