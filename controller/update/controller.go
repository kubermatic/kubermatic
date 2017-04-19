package update

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/resources"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/extensions/etcd"
	"github.com/kubermatic/api/provider/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	k "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// Controller represents an update controller
type Controller struct {
	Client              k.Interface
	TprClient           extensions.Clientset
	EtcdClusterClient   etcd.Clientset
	MasterResourcesPath string
	OverwriteHost       string
	DC                  string
	Versions            map[string]*api.MasterVersion
	Updates             []api.MasterUpdate
	DepStore            cache.Indexer
	EtcdClusterStore    cache.Indexer
}

// Sync determines the current update state, and advances to the next phase as required
func (u *Controller) Sync(c *api.Cluster) (*api.Cluster, error) {
	v, found := u.Versions[c.Spec.MasterVersion]
	if !found {
		return nil, fmt.Errorf("unknown target master version %q", c.Spec.MasterVersion)
	}

	switch c.Status.MasterUpdatePhase {
	case api.StartMasterUpdatePhase:
		return u.updateDeployment(c, []string{v.EtcdOperatorDeploymentYaml}, v, api.EtcdOperatorUpdatePhase)
	case api.EtcdOperatorUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"etcd-operator"}, api.StartMasterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateEtcdCluster(c, []string{v.EtcdClusterYaml}, v, api.EtcdClusterUpdatePhase)
	case api.EtcdClusterUpdatePhase:
		c, ready, err := u.waitForEtcdCluster(c, []string{"etcd-cluster"}, api.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateDeployment(c, []string{v.ApiserverDeploymentYaml}, v, api.APIServerMasterUpdatePhase)
	case api.APIServerMasterUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"apiserver"}, api.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}
		return u.updateDeployment(c, []string{v.ControllerDeploymentYaml, v.SchedulerDeploymentYaml}, v, api.ControllersMasterUpdatePhase)
	case api.ControllersMasterUpdatePhase:
		c, ready, err := u.waitForDeployments(c, []string{"controller-manager", "scheduler"}, api.EtcdClusterUpdatePhase)
		if !ready || err != nil {
			return c, err
		}

		c.Status.MasterUpdatePhase = ""
		c.Status.Phase = api.RunningClusterStatusPhase
		c.Status.LastTransitionTime = time.Now()
		return c, nil
	}

	// this should never happen: we don't know the phase
	glog.Errorf("Unknown cluster %q update phase: %v", c.Metadata.Name, c.Status.MasterUpdatePhase)
	c.Status.MasterUpdatePhase = ""
	c.Status.Phase = api.RunningClusterStatusPhase
	c.Status.LastTransitionTime = time.Now()
	return c, nil
}

func (u *Controller) updateDeployment(c *api.Cluster, yamlFiles []string, masterVersion *api.MasterVersion, nextPhase api.MasterUpdatePhase) (*api.Cluster, error) {
	for _, yamlFile := range yamlFiles {
		dep, err := resources.LoadDeploymentFile(c, masterVersion, u.MasterResourcesPath, u.DC, yamlFile)
		if err != nil {
			return nil, err
		}

		ns := kubernetes.NamespaceName(c.Metadata.Name)
		_, err = u.Client.ExtensionsV1beta1().Deployments(ns).Update(dep)
		if errors.IsNotFound(err) {
			glog.Errorf("expected an %s deployment, but didn't find any for cluster %v. Creating a new one.", dep.Name, c.Metadata.Name)
			_, err = u.Client.ExtensionsV1beta1().Deployments(ns).Create(dep)
			if err != nil {
				return nil, fmt.Errorf("failed to re-create deployment for %s: %v", dep.Name, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to update deployment for %s: %v", dep.Name, err)
		}
	}

	c.Status.MasterUpdatePhase = nextPhase
	return c, nil
}

func (u *Controller) updateEtcdCluster(c *api.Cluster, yamlFiles []string, masterVersion *api.MasterVersion, nextPhase api.MasterUpdatePhase) (*api.Cluster, error) {
	for _, yamlFile := range yamlFiles {
		etcd, err := resources.LoadEtcdClusterFile(masterVersion, u.MasterResourcesPath, yamlFile)
		if err != nil {
			return nil, err
		}

		_, err = u.EtcdClusterClient.Cluster(fmt.Sprintf("cluster-%s", c.Metadata.Name)).Update(etcd)
		if errors.IsNotFound(err) {
			glog.Errorf("expected an %s etcd cluster, but didn't find any for cluster %v. Creating a new one.", etcd.Metadata.Name, c.Metadata.Name)
			_, err = u.EtcdClusterClient.Cluster(fmt.Sprintf("cluster-%s", c.Metadata.Name)).Create(etcd)
			if err != nil {
				return nil, fmt.Errorf("failed to re-create etcd cluster for %s: %v", etcd.Metadata.Name, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to update etcd cluster  for %s: %v", etcd.Metadata.Name, err)
		}
	}

	c.Status.MasterUpdatePhase = nextPhase
	return c, nil
}

func (u *Controller) waitForEtcdCluster(c *api.Cluster, names []string, fallbackPhase api.MasterUpdatePhase) (*api.Cluster, bool, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)

	for _, name := range names {
		obj, exists, err := u.EtcdClusterStore.GetByKey(fmt.Sprintf("%s/%s", ns, name))
		if err != nil {
			return nil, false, err
		}
		if !exists {
			glog.Errorf("expected an %s etcd cluster, but didn't find any for cluster %v.", name, c.Metadata.Name)
			c.Status.MasterUpdatePhase = fallbackPhase
			return c, false, nil
		}
		etcd := obj.(*etcd.Cluster)
		//Ensure the etcd quorum
		if etcd.Spec.Size/2+1 >= etcd.Status.Size {
			return nil, false, nil
		}
	}
	return c, true, nil
}

func (u *Controller) waitForDeployments(c *api.Cluster, names []string, fallbackPhase api.MasterUpdatePhase) (*api.Cluster, bool, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)

	for _, name := range names {
		dep, exists, err := u.DepStore.GetByKey(fmt.Sprintf("%s/%s", ns, name))
		if err != nil {
			return nil, false, err
		}
		if !exists {
			glog.Errorf("expected an %s deployment, but didn't find any for cluster %v.", name, c.Metadata.Name)
			c.Status.MasterUpdatePhase = fallbackPhase
			return c, false, nil
		}
		if !healthyDep(dep.(*v1beta1.Deployment)) {
			return nil, false, nil
		}
	}
	return c, true, nil
}

// healthyDep is true if >= 90% of the expected pods are ready
func healthyDep(dep *v1beta1.Deployment) bool {
	return float64(dep.Status.UpdatedReplicas) >= 0.9*float64(*dep.Spec.Replicas)
}
