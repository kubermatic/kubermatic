package update

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/controller/resources"
	crdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	"github.com/kubermatic/kubermatic/api/provider/kubernetes"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientkubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Interface is a interface for a update controller
type Interface interface {
	Sync(*api.Cluster) (*api.Cluster, error)
}

// New returns a update controller
func New(
	kubeClient clientkubernetes.Interface,
	crdClient crdclient.Interface,
	masterResourcesPath,
	overwriteHost,
	dc string,
	versions map[string]*api.MasterVersion,
	updates []api.MasterUpdate,
	depStore cache.Indexer,
	etcdClusterStore cache.Indexer,
) Interface {
	return &controller{
		client:              kubeClient,
		crdClient:           crdClient,
		masterResourcesPath: masterResourcesPath,
		overwriteHost:       overwriteHost,
		dc:                  dc,
		versions:            versions,
		updates:             updates,
		depStore:            depStore,
		etcdClusterStore:    etcdClusterStore,
	}
}

// Controller represents an update controller
type controller struct {
	client              clientkubernetes.Interface
	crdClient           crdclient.Interface
	masterResourcesPath string
	overwriteHost       string
	dc                  string
	versions            map[string]*api.MasterVersion
	updates             []api.MasterUpdate
	depStore            cache.Indexer
	etcdClusterStore    cache.Indexer
}

// Sync determines the current update state, and advances to the next phase as required
func (u *controller) Sync(c *api.Cluster) (*api.Cluster, error) {
	v, found := u.versions[c.Spec.MasterVersion]
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

func (u *controller) updateDeployment(c *api.Cluster, yamlFiles []string, masterVersion *api.MasterVersion, nextPhase api.MasterUpdatePhase) (*api.Cluster, error) {
	for _, yamlFile := range yamlFiles {
		dep, err := resources.LoadDeploymentFile(c, masterVersion, u.masterResourcesPath, u.dc, yamlFile)
		if err != nil {
			return nil, err
		}

		ns := kubernetes.NamespaceName(c.Metadata.Name)
		_, err = u.client.ExtensionsV1beta1().Deployments(ns).Update(dep)
		if errors.IsNotFound(err) {
			glog.Errorf("expected an %s deployment, but didn't find any for cluster %v. Creating a new one.", dep.Name, c.Metadata.Name)
			_, err = u.client.ExtensionsV1beta1().Deployments(ns).Create(dep)
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

func (u *controller) updateEtcdCluster(c *api.Cluster, yamlFiles []string, masterVersion *api.MasterVersion, nextPhase api.MasterUpdatePhase) (*api.Cluster, error) {
	for _, yamlFile := range yamlFiles {
		newEtcd, err := resources.LoadEtcdClusterFile(masterVersion, u.masterResourcesPath, yamlFile)
		if err != nil {
			return nil, err
		}
		ns := fmt.Sprintf("cluster-%s", c.Metadata.Name)
		var oldEtcd *etcdoperatorv1beta2.EtcdCluster
		oldEtcd, err = u.crdClient.EtcdoperatorV1beta2().EtcdClusters(ns).Get("etcd-cluster", metav1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("failed to get current etcd cluster for %s: %v", newEtcd.ObjectMeta.Name, err)
			glog.Error(err)
			return nil, err
		}

		oldEtcd.Spec.Version = newEtcd.Spec.Version
		_, err = u.crdClient.EtcdoperatorV1beta2().EtcdClusters(ns).Update(oldEtcd)
		if err != nil {
			err = fmt.Errorf("failed to update etcd cluster for %s: %v", newEtcd.ObjectMeta.Name, err)
			glog.Error(err)
			return nil, err
		}
	}
	c.Status.MasterUpdatePhase = nextPhase
	return c, nil
}

func (u *controller) waitForEtcdCluster(c *api.Cluster, names []string, fallbackPhase api.MasterUpdatePhase) (*api.Cluster, bool, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)

	for _, name := range names {
		obj, exists, err := u.etcdClusterStore.GetByKey(fmt.Sprintf("%s/%s", ns, name))
		if err != nil {
			return nil, false, err
		}
		if !exists {
			glog.Errorf("expected an %s etcd cluster, but didn't find any for cluster %v.", name, c.Metadata.Name)
			c.Status.MasterUpdatePhase = fallbackPhase
			return c, false, nil
		}
		etcd := obj.(*etcdoperatorv1beta2.EtcdCluster)
		//Ensure the etcd quorum
		if etcd.Spec.Size/2+1 >= etcd.Status.Size {
			return nil, false, nil
		}
	}
	return c, true, nil
}

func (u *controller) waitForDeployments(c *api.Cluster, names []string, fallbackPhase api.MasterUpdatePhase) (*api.Cluster, bool, error) {
	ns := kubernetes.NamespaceName(c.Metadata.Name)

	for _, name := range names {
		dep, exists, err := u.depStore.GetByKey(fmt.Sprintf("%s/%s", ns, name))
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
