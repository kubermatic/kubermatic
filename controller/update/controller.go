package update

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
	Versions            map[string]*api.MasterVersion
	Updates             []api.MasterUpdate
}

func (u *UpdateController) Sync(c *api.Cluster) (*api.Cluster, error) {

	switch c.Status.MasterUpdatePhase {
	case api.StartMasterUpdatePhase:
		c, err := u.updateEtcd(c)
		if err != nil {

		}
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
	v, found := u.Versions[c.Spec.MasterVersion]
	if !found {
		return nil, fmt.Errorf("unknown target master version %q", c.Spec.MasterVersion)
	}

	etcdDep, err := resources.LoadDeploymentFile(c, v, u.MasterResourcesPath, u.OverwriteHost, u.DC)
	if err != nil {
		return nil, err
	}

	ns := kubernetes.NamespaceName(c.Metadata.User, c.Metadata.Name)

	_, err = u.Client.ExtensionsV1beta1().Deployments(ns).Update(etcdDep)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment for etcd: %v", err)
	}

	c.Status.MasterUpdatePhase = api.EtcdMasterUpdatePhase

	return c, nil
}
