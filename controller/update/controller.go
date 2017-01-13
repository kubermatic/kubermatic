package cluster

import "github.com/kubermatic/api"
import "github.com/kubermatic/api/controller/resources"
import "k8s.io/client-go/kubernetes"

type UpdateController struct {
	Client              kubernetes.Interface
	MasterResourcesPath string
	OverwriteHost       string
	DC                  string
}

func (u *UpdateController) doUpdate(c *api.Cluster) (*api.Cluster, error) {

	switch c.Status.MasterUpdatePhase {
	case api.StartMasterUpdatePhase:

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

	etcdDep, err := resources.LoadDeploymentFile(c, u.MasterResourcesPath, u.OverwriteHost, u.DC, s)
	if err != nil {
		return nil, err
	}

	// push deployment to api

	// switch phase

	return c, nil

}
