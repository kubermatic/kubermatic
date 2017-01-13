package cluster

import (
	"fmt"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/resources"
	"github.com/kubermatic/api/provider/kubernetes"
	k "k8s.io/client-go/kubernetes"
)

type UpdateController struct {
	Client              k.Interface
	MasterResourcesPath string
	OverwriteHost       string
	DC                  string
	Versions            map[string]api.MasterVersion
}

func (u *UpdateController) doUpdate(c *api.Cluster) (*api.Cluster, error) {

	switch c.Status.MasterUpdatePhase {
	case api.StartMasterUpdatePhase:
		return u.updateEtcd(c)
	case api.EtcdMasterUpdatePhase:
		//wait for etcd, update api server
	case api.APIServerMasterUpdatePhase:
		//wait for api, update controllers
	case api.ControllersMasterUpdatePhase:
		//wait for controllers

	}

	return c, nil

}

func (u *UpdateController) updateEtcd(c *api.Cluster) (*api.Cluster, error) {

	v := u.Versions[c.Spec.TargetMasterVersion]

	etcdDep, err := resources.LoadDeploymentFile(c, v, u.MasterResourcesPath, u.OverwriteHost, u.DC)
	if err != nil {
		return nil, err
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	_, err = u.Client.ExtensionsV1beta1().Deployments(ns).Create(etcdDep)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment for etcd: %v", err)
	}

	c.Status.MasterUpdatePhase = api.EtcdMasterUpdatePhase

	return c, nil

}
